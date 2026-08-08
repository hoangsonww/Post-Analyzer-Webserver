output "cluster_name" {
  value = azurerm_kubernetes_cluster.main.name
}

output "kubeconfig_command" {
  value = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_kubernetes_cluster.main.name}"
}

output "acr_login_server" {
  value = azurerm_container_registry.main.login_server
}

output "postgres_fqdn" {
  value = azurerm_postgresql_flexible_server.main.fqdn
}

output "redis_hostname" {
  value = azurerm_redis_cache.main.hostname
}

output "cdn_endpoint_hostname" {
  description = "Front Door endpoint hostname (only set when enable_cdn = true)"
  value       = var.enable_cdn ? azurerm_cdn_frontdoor_endpoint.main[0].host_name : null
}
