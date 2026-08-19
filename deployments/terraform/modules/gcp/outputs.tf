output "cluster_name" {
  value = google_container_cluster.main.name
}

output "cluster_endpoint" {
  value = google_container_cluster.main.endpoint
}

output "kubeconfig_command" {
  description = "Run this to point kubectl at the new cluster"
  value       = "gcloud container clusters get-credentials ${google_container_cluster.main.name} --region ${var.region} --project ${var.project_id}"
}

output "postgres_connection_name" {
  value = google_sql_database_instance.postgres.connection_name
}

output "postgres_private_ip" {
  value = google_sql_database_instance.postgres.private_ip_address
}

output "redis_endpoint" {
  value = google_redis_instance.main.host
}

output "artifact_registry_repository_urls" {
  value = {
    for k, v in google_artifact_registry_repository.services :
    k => "${v.location}-docker.pkg.dev/${var.project_id}/${v.repository_id}"
  }
}

output "cdn_ip_address" {
  description = "Public IP of the Cloud CDN forwarding rule (only set when enable_cdn = true)"
  value       = var.enable_cdn ? google_compute_global_forwarding_rule.cdn[0].ip_address : null
}
