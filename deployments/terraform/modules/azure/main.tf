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
