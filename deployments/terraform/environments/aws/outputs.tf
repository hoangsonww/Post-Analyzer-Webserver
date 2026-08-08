output "cluster_name" { value = module.post_analyzer.cluster_name }
output "kubeconfig_command" { value = module.post_analyzer.kubeconfig_command }
output "postgres_endpoint" { value = module.post_analyzer.postgres_endpoint }
output "redis_endpoint" { value = module.post_analyzer.redis_endpoint }
output "ecr_repository_urls" { value = module.post_analyzer.ecr_repository_urls }
