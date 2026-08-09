# Azure module: AKS instead of the local kind cluster, Azure Database for
# PostgreSQL Flexible Server instead of the in-cluster Postgres
# Deployment, Azure Cache for Redis instead of the Redis Deployment, and
# ACR for the three service images.

terraform {
  required_version = ">= 1.5"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

locals {
  name = "${var.project}-${var.environment}"
  tags = {
    project     = var.project
    environment = var.environment
    managed_by  = "terraform"
  }
}

resource "azurerm_resource_group" "main" {
  name     = "${local.name}-rg"
  location = var.location
  tags     = local.tags
}

resource "azurerm_virtual_network" "main" {
  name                = "${local.name}-vnet"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  address_space       = ["10.30.0.0/16"]
  tags                = local.tags
}

resource "azurerm_subnet" "aks" {
  name                 = "${local.name}-aks-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.30.1.0/24"]
}

resource "azurerm_subnet" "data" {
  name                 = "${local.name}-data-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.30.2.0/24"]
  delegation {
    name = "postgres-flexible-server"
    service_delegation {
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

# --- AKS ------------------------------------------------------------------

resource "azurerm_kubernetes_cluster" "main" {
  name                = local.name
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  dns_prefix          = local.name
  kubernetes_version  = var.kubernetes_version

  default_node_pool {
    name           = "default"
    node_count     = var.node_count
    vm_size        = var.node_vm_size
    vnet_subnet_id = azurerm_subnet.aks.id
  }

  identity {
    type = "SystemAssigned"
  }

  tags = local.tags
}

# --- ACR --------------------------------------------------------------

resource "azurerm_container_registry" "main" {
  name                = replace("${local.name}acr", "-", "")
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Standard"
  admin_enabled       = false
  tags                = local.tags
}

resource "azurerm_role_assignment" "aks_pull" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.main.kubelet_identity[0].object_id
}

# --- Azure Database for PostgreSQL Flexible Server -----------------------

resource "azurerm_private_dns_zone" "postgres" {
  name                = "${local.name}.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  name                  = "${local.name}-postgres-link"
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = azurerm_virtual_network.main.id
  resource_group_name   = azurerm_resource_group.main.name
}

resource "azurerm_postgresql_flexible_server" "main" {
  name                   = "${local.name}-postgres"
  resource_group_name    = azurerm_resource_group.main.name
  location               = azurerm_resource_group.main.location
  version                = "16"
  delegated_subnet_id    = azurerm_subnet.data.id
  private_dns_zone_id    = azurerm_private_dns_zone.postgres.id
  administrator_login    = "postgres"
  administrator_password = var.db_admin_password
  sku_name               = var.db_sku_name
  storage_mb             = 32768
  zone                   = "1"

  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgres]
  tags       = local.tags
}

resource "azurerm_postgresql_flexible_server_database" "postanalyzer" {
  name      = "postanalyzer"
  server_id = azurerm_postgresql_flexible_server.main.id
  collation = "en_US.utf8"
  charset   = "utf8"
}

# --- Azure Cache for Redis ---------------------------------------------

resource "azurerm_redis_cache" "main" {
  name                 = "${local.name}-redis"
  location             = azurerm_resource_group.main.location
  resource_group_name  = azurerm_resource_group.main.name
  capacity             = var.redis_capacity
  family               = "C"
  sku_name             = "Basic"
  non_ssl_port_enabled = false
  tags                 = local.tags
}

# --- CDN (Azure Front Door, edge cache in front of the AKS ingress) ------
#
# Off by default (var.enable_cdn = false): the real origin is the public
# IP/hostname the AKS ingress-nginx controller's LoadBalancer Service is
# assigned once deployments/k8s/ is applied to this cluster — that
# doesn't exist until after this module's apply, so wiring it in is a
# two-phase deploy (apply this module -> deploy ingress-nginx -> read its
# LoadBalancer IP -> re-apply with enable_cdn=true and that hostname).
# Static assets get edge-cached; /api/* uses caching_enabled = false so
# auth/ABAC-gated responses are never cached at the edge.
resource "azurerm_cdn_frontdoor_profile" "main" {
  count               = var.enable_cdn ? 1 : 0
  name                = "${local.name}-fd"
  resource_group_name = azurerm_resource_group.main.name
  sku_name            = "Standard_AzureFrontDoor"
  tags                = local.tags
}

resource "azurerm_cdn_frontdoor_endpoint" "main" {
  count                    = var.enable_cdn ? 1 : 0
  name                     = "${local.name}-fd-endpoint"
  cdn_frontdoor_profile_id = azurerm_cdn_frontdoor_profile.main[0].id
  tags                     = local.tags
}

resource "azurerm_cdn_frontdoor_origin_group" "main" {
  count                    = var.enable_cdn ? 1 : 0
  name                     = "${local.name}-origin-group"
  cdn_frontdoor_profile_id = azurerm_cdn_frontdoor_profile.main[0].id

  load_balancing {
    sample_size                 = 4
    successful_samples_required = 3
  }

  health_probe {
    protocol            = "Http"
    request_type        = "GET"
    path                = "/health"
    interval_in_seconds = 30
  }
}

resource "azurerm_cdn_frontdoor_origin" "main" {
  count                          = var.enable_cdn ? 1 : 0
  name                           = "${local.name}-origin"
  cdn_frontdoor_origin_group_id  = azurerm_cdn_frontdoor_origin_group.main[0].id
  host_name                      = var.cdn_origin_hostname
  origin_host_header             = var.cdn_origin_hostname
  http_port                      = 80
  https_port                     = 443
  certificate_name_check_enabled = false
}

# /api/* — no cache block, so Front Door never caches auth/ABAC-gated
# responses at the edge. More specific pattern than the default route
# below, so it wins for API traffic.
resource "azurerm_cdn_frontdoor_route" "api" {
  count                         = var.enable_cdn ? 1 : 0
  name                          = "${local.name}-route-api"
  cdn_frontdoor_endpoint_id     = azurerm_cdn_frontdoor_endpoint.main[0].id
  cdn_frontdoor_origin_group_id = azurerm_cdn_frontdoor_origin_group.main[0].id
  cdn_frontdoor_origin_ids      = [azurerm_cdn_frontdoor_origin.main[0].id]
  supported_protocols           = ["Http", "Https"]
  patterns_to_match             = ["/api/*"]
  forwarding_protocol           = "HttpOnly"
  link_to_default_domain        = true
}

# Everything else (static UI assets, dashboard, home page) — cached at
# the edge.
resource "azurerm_cdn_frontdoor_route" "static" {
  count                         = var.enable_cdn ? 1 : 0
  name                          = "${local.name}-route-static"
  cdn_frontdoor_endpoint_id     = azurerm_cdn_frontdoor_endpoint.main[0].id
  cdn_frontdoor_origin_group_id = azurerm_cdn_frontdoor_origin_group.main[0].id
  cdn_frontdoor_origin_ids      = [azurerm_cdn_frontdoor_origin.main[0].id]
  supported_protocols           = ["Http", "Https"]
  patterns_to_match             = ["/*"]
  forwarding_protocol           = "HttpOnly"
  link_to_default_domain        = true

  cache {
    query_string_caching_behavior = "IgnoreQueryString"
    compression_enabled           = true
  }
}
