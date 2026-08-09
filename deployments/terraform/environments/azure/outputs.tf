output "cluster_name" { value = module.post_analyzer.cluster_name }
output "kubeconfig_command" { value = module.post_analyzer.kubeconfig_command }
output "acr_login_server" { value = module.post_analyzer.acr_login_server }
output "postgres_fqdn" { value = module.post_analyzer.postgres_fqdn }
output "redis_hostname" { value = module.post_analyzer.redis_hostname }
output "cdn_endpoint_hostname" { value = module.post_analyzer.cdn_endpoint_hostname }
