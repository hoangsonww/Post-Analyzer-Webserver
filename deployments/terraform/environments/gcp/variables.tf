variable "project_id" {
  description = "GCP project ID to deploy into (must already exist and be linked to a billing account)"
  type        = string
}

variable "project" {
  type    = string
  default = "post-analyzer"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "enable_cdn" {
  description = "Provision a Cloud CDN-fronted load balancer. Requires cdn_origin_ip — see modules/gcp/main.tf."
  type        = bool
  default     = false
}

variable "cdn_origin_ip" {
  description = "External IP of the ingress-nginx LoadBalancer Service, known only after the K8s ingress is deployed to this cluster. Only required when enable_cdn = true."
  type        = string
  default     = ""
}

variable "cdn_origin_port" {
  type    = number
  default = 80
}
