# GCP module: provisions the managed equivalents of what docker-compose
# runs locally — a GKE cluster to run deployments/k8s/ on, Cloud SQL
# Postgres instead of the in-cluster Postgres Deployment, Memorystore
# instead of the Redis Deployment, and Artifact Registry repos to push
# the six service images to (see deployments/k8s/overlays/*/kustomization.yaml
# `images:` for where those repo URLs plug in). Mirrors the shape of
# modules/aws as closely as GCP's resource model allows.

terraform {
  required_version = ">= 1.5"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

locals {
  name     = "${var.project}-${var.environment}"
  services = toset(["gateway", "postsvc", "authsvc", "analytics-consumer", "reanalysis-worker", "notification-consumer"])
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# --- Required APIs ------------------------------------------------------

resource "google_project_service" "required" {
  for_each = toset([
    "container.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "artifactregistry.googleapis.com",
    "servicenetworking.googleapis.com",
    "compute.googleapis.com",
  ])
  project            = var.project_id
  service            = each.key
  disable_on_destroy = false
}

# --- Networking -----------------------------------------------------------

resource "google_compute_network" "main" {
  name                    = local.name
  auto_create_subnetworks = false
  depends_on              = [google_project_service.required]
}

resource "google_compute_subnetwork" "main" {
  name          = "${local.name}-subnet"
  network       = google_compute_network.main.id
  ip_cidr_range = var.vpc_cidr
  region        = var.region

  secondary_ip_range {
    range_name    = "${local.name}-pods"
    ip_cidr_range = var.pods_cidr
  }
  secondary_ip_range {
    range_name    = "${local.name}-services"
    ip_cidr_range = var.services_cidr
  }
}

# Cloud Router + Cloud NAT so private GKE nodes (no public IPs) can still
# pull public images and reach the internet.
resource "google_compute_router" "main" {
  name    = "${local.name}-router"
  network = google_compute_network.main.id
  region  = var.region
}

resource "google_compute_router_nat" "main" {
  name                               = "${local.name}-nat"
  router                             = google_compute_router.main.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

# Private services access (VPC peering) — required for Cloud SQL and
# Memorystore to get a private IP inside this VPC instead of a public one.
resource "google_compute_global_address" "private_service_range" {
  name          = "${local.name}-psa"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.main.id
}

resource "google_service_networking_connection" "private_service" {
  network                 = google_compute_network.main.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_service_range.name]
  depends_on              = [google_project_service.required]
}

# --- GKE --------------------------------------------------------------

resource "google_container_cluster" "main" {
  name     = local.name
  location = var.region # regional cluster: nodes spread across zones for HA

  network    = google_compute_network.main.id
  subnetwork = google_compute_subnetwork.main.id

  # Node pool managed separately below so it can be resized/upgraded
  # without recreating the cluster control plane.
  remove_default_node_pool = true
  initial_node_count       = 1

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {
    cluster_secondary_range_name  = "${local.name}-pods"
    services_secondary_range_name = "${local.name}-services"
  }

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false # keep the API server publicly reachable, same posture as AWS module's endpoint_public_access = true
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  release_channel {
    channel = var.kubernetes_version
  }

  deletion_protection = false

  depends_on = [google_project_service.required, google_compute_router_nat.main]
}

resource "google_service_account" "node" {
  account_id   = "${local.name}-gke-node"
  display_name = "${local.name} GKE node service account"
}

resource "google_project_iam_member" "node_roles" {
  for_each = toset([
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/artifactregistry.reader",
  ])
  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.node.email}"
}

resource "google_container_node_pool" "main" {
  name     = "${local.name}-nodes"
  location = var.region
  cluster  = google_container_cluster.main.name

  initial_node_count = var.node_desired_size
  autoscaling {
    min_node_count = 1
    max_node_count = var.node_desired_size * 2
  }

  node_config {
    machine_type    = var.node_machine_type
    service_account = google_service_account.node.email
    oauth_scopes    = ["https://www.googleapis.com/auth/cloud-platform"]
  }
}

# --- Cloud SQL Postgres -----------------------------------------------

resource "google_sql_database_instance" "postgres" {
  name             = "${local.name}-postgres"
  database_version = "POSTGRES_16"
  region           = var.region

  settings {
    tier = var.db_tier
    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.main.id
    }
    availability_type = var.environment == "prod" ? "REGIONAL" : "ZONAL"
    backup_configuration {
      enabled = var.environment == "prod"
    }
  }

  deletion_protection = false
  depends_on          = [google_service_networking_connection.private_service]
}

resource "google_sql_database" "main" {
  name     = "postanalyzer"
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_user" "main" {
  name     = "postgres"
  instance = google_sql_database_instance.postgres.name
  password = var.db_password
}

# --- Memorystore Redis --------------------------------------------------

resource "google_redis_instance" "main" {
  name           = "${local.name}-redis"
  tier           = var.redis_tier
  memory_size_gb = var.redis_memory_size_gb
  region         = var.region

  authorized_network = google_compute_network.main.id
  connect_mode       = "PRIVATE_SERVICE_ACCESS"
  redis_version      = "REDIS_7_2"

  depends_on = [google_service_networking_connection.private_service]
}

# --- Artifact Registry ----------------------------------------------------

resource "google_artifact_registry_repository" "services" {
  for_each      = local.services
  location      = var.region
  repository_id = "${var.project}-${each.key}"
  format        = "DOCKER"

  depends_on = [google_project_service.required]
}

# --- CDN (edge cache in front of the ingress-nginx LoadBalancer Service) --
#
# Off by default (var.enable_cdn = false): the real origin is the external
# IP that Kubernetes' ingress-nginx controller creates when
# deployments/k8s/overlays/<env> is applied to this cluster — that IP
# doesn't exist until *after* this module's apply, so wiring it in is
# inherently a two-phase deploy (apply this module -> deploy ingress-nginx
# -> `kubectl get svc -n ingress-nginx` for the external IP -> re-apply
# with enable_cdn=true and that IP). Uses an INTERNET_IP_PORT network
# endpoint group, which is GCP's mechanism for fronting an
# already-load-balanced non-GCP-native origin with Cloud CDN. Static
# assets (/assets/*, /dashboard, /home.html) get edge-cached; /api/* skips
# the CDN-enabled default backend so auth/ABAC-gated responses are never
# cached at the edge — mirroring the CachingDisabled behavior of the AWS
# module's /api/* ordered_cache_behavior.
resource "google_compute_global_network_endpoint_group" "origin" {
  count                 = var.enable_cdn ? 1 : 0
  name                  = "${local.name}-origin-neg"
  network_endpoint_type = "INTERNET_IP_PORT"
  default_port          = var.cdn_origin_port
}

resource "google_compute_global_network_endpoint" "origin" {
  count                         = var.enable_cdn ? 1 : 0
  global_network_endpoint_group = google_compute_global_network_endpoint_group.origin[0].id
  ip_address                    = var.cdn_origin_ip
  port                          = var.cdn_origin_port
}

resource "google_compute_backend_service" "cdn" {
  count                 = var.enable_cdn ? 1 : 0
  name                  = "${local.name}-backend"
  protocol              = "HTTP"
  timeout_sec           = 30
  enable_cdn            = true
  load_balancing_scheme = "EXTERNAL"

  backend {
    group = google_compute_global_network_endpoint_group.origin[0].id
  }

  cdn_policy {
    cache_mode  = "CACHE_ALL_STATIC"
    client_ttl  = 3600
    default_ttl = 3600
    max_ttl     = 86400
  }
}

resource "google_compute_url_map" "cdn" {
  count           = var.enable_cdn ? 1 : 0
  name            = "${local.name}-urlmap"
  default_service = google_compute_backend_service.cdn[0].id

  path_matcher {
    name            = "no-cache-api"
    default_service = google_compute_backend_service.cdn[0].id
    path_rule {
      paths   = ["/api/*"]
      service = google_compute_backend_service.cdn[0].id
    }
  }
}

resource "google_compute_target_http_proxy" "cdn" {
  count   = var.enable_cdn ? 1 : 0
  name    = "${local.name}-http-proxy"
  url_map = google_compute_url_map.cdn[0].id
}

resource "google_compute_global_forwarding_rule" "cdn" {
  count                 = var.enable_cdn ? 1 : 0
  name                  = "${local.name}-fwd-rule"
  target                = google_compute_target_http_proxy.cdn[0].id
  port_range            = "80"
  load_balancing_scheme = "EXTERNAL"
}
