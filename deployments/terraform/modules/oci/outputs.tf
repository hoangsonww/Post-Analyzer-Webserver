output "cluster_id" {
  value = oci_containerengine_cluster.main.id
}

output "kubeconfig_command" {
  value = "oci ce cluster create-kubeconfig --cluster-id ${oci_containerengine_cluster.main.id} --file $HOME/.kube/config --region ${var.region} --token-version 2.0.0"
}

output "postgres_db_system_id" {
  value = oci_psql_db_system.main.id
}

output "container_repository_names" {
  value = { for k, v in oci_artifacts_container_repository.services : k => v.display_name }
}
