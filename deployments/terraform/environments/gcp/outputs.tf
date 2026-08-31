output "cluster_name" { value = module.post_analyzer.cluster_name }
output "kubeconfig_command" { value = module.post_analyzer.kubeconfig_command }
output "postgres_connection_name" { value = module.post_analyzer.postgres_connection_name }
output "postgres_private_ip" { value = module.post_analyzer.postgres_private_ip }
output "redis_endpoint" { value = module.post_analyzer.redis_endpoint }
output "artifact_registry_repository_urls" { value = module.post_analyzer.artifact_registry_repository_urls }
output "cdn_ip_address" { value = module.post_analyzer.cdn_ip_address }
