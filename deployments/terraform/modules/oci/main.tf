# OCI module: OKE instead of the local kind cluster, the OCI Postgres
# managed service instead of the in-cluster Postgres Deployment, OCI
# Cache (Redis-compatible) instead of the Redis Deployment, and OCIR
# (implicit per-region, just needs a repository resource per image) for
# the three service images.

terraform {
  required_version = ">= 1.5"
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 5.0"
    }
  }
}

provider "oci" {
  region = var.region
}

locals {
  name = "${var.project}-${var.environment}"
  freeform_tags = {
    project     = var.project
    environment = var.environment
    managed_by  = "terraform"
  }
}

# --- Networking -------------------------------------------------------

resource "oci_core_vcn" "main" {
  compartment_id = var.compartment_id
  cidr_blocks    = ["10.40.0.0/16"]
  display_name   = "${local.name}-vcn"
  freeform_tags  = local.freeform_tags
}

resource "oci_core_internet_gateway" "main" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.main.id
  display_name   = "${local.name}-igw"
}

resource "oci_core_default_route_table" "main" {
  manage_default_resource_id = oci_core_vcn.main.default_route_table_id
  route_rules {
    destination       = "0.0.0.0/0"
    network_entity_id = oci_core_internet_gateway.main.id
  }
}

resource "oci_core_subnet" "nodes" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.main.id
  cidr_block     = "10.40.1.0/24"
  display_name   = "${local.name}-nodes"
}

resource "oci_core_subnet" "db" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.main.id
  cidr_block     = "10.40.2.0/24"
  display_name   = "${local.name}-db"
}

# --- OKE ------------------------------------------------------------------

resource "oci_containerengine_cluster" "main" {
  compartment_id     = var.compartment_id
  name               = local.name
  vcn_id             = oci_core_vcn.main.id
  kubernetes_version = var.kubernetes_version

  endpoint_config {
    is_public_ip_enabled = true
    subnet_id            = oci_core_subnet.nodes.id
  }

  freeform_tags = local.freeform_tags
}

resource "oci_containerengine_node_pool" "main" {
  compartment_id     = var.compartment_id
  cluster_id         = oci_containerengine_cluster.main.id
  name               = "${local.name}-pool"
  kubernetes_version = var.kubernetes_version
  node_shape         = var.node_shape

  node_shape_config {
    ocpus         = 2
    memory_in_gbs = 16
  }

  node_config_details {
    size = var.node_pool_size
    placement_configs {
      availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
      subnet_id           = oci_core_subnet.nodes.id
    }
  }

  node_source_details {
    source_type = "IMAGE"
    image_id    = data.oci_core_images.oke_images.images[0].id
  }

  freeform_tags = local.freeform_tags
}

data "oci_identity_availability_domains" "ads" {
  compartment_id = var.compartment_id
}

data "oci_core_images" "oke_images" {
  compartment_id           = var.compartment_id
  operating_system         = "Oracle Linux"
  operating_system_version = "8"
  shape                    = var.node_shape
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

# --- OCI Postgres -------------------------------------------------------

resource "oci_psql_db_system" "main" {
  compartment_id = var.compartment_id
  display_name   = "${local.name}-postgres"
  db_version     = "16"
  shape          = "PostgreSQL.VM.Standard.E4.Flex.2.32"

  network_details {
    subnet_id = oci_core_subnet.db.id
  }

  credentials {
    username = "postgres"
    password_details {
      password_type = "PLAIN_TEXT"
      password      = var.db_admin_password
    }
  }

  storage_details {
    is_regionally_durable = false
    availability_domain   = data.oci_identity_availability_domains.ads.availability_domains[0].name
    system_type           = "OCI_OPTIMIZED_STORAGE"
  }

  # config_id intentionally omitted: leaves OCI's system default PostgreSQL
  # 16 configuration in place. Pass a specific oci_psql_configuration OCID
  # here (`oci psql configuration list`) to tune server parameters.

  freeform_tags = local.freeform_tags
}

# --- OCIR (container registry) -----------------------------------------

resource "oci_artifacts_container_repository" "services" {
  for_each       = toset(["gateway", "postsvc", "authsvc", "analytics-consumer", "reanalysis-worker", "notification-consumer"])
  compartment_id = var.compartment_id
  display_name   = "${var.project}/${each.key}"
  is_public      = false
}

# --- CDN ------------------------------------------------------------------
#
# Deliberately no CDN resource here, unlike the AWS (CloudFront) and Azure
# (Front Door) modules: OCI does not ship a first-party edge-cache CDN
# product with a Terraform resource in the `oci` provider — its edge-layer
# offerings are OCI WAF (`oci_waf_web_app_firewall`, security/rate-limiting
# at the edge, not caching) and third-party CDN partners (e.g. Cloudflare,
# Akamai) provisioned via *their* Terraform providers in front of the OKE
# ingress LoadBalancer created by deployments/k8s/. If this module is
# picked for a real deployment and edge caching is required, the
# recommended path is: stand up the OCI Load Balancer via ingress-nginx as
# usual, then front it with a separate CDN provider's Terraform module —
# not something to fake here with an unsupported resource.
