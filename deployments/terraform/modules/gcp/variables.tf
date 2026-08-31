variable "project_id" {
  description = "GCP project ID to deploy into (must already exist and be linked to a billing account)"
  type        = string
}

variable "project" {
  description = "Project name, used as a resource name prefix"
  type        = string
  default     = "post-analyzer"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "vpc_cidr" {
  description = "Primary CIDR range for the GKE subnet"
  type        = string
  default     = "10.30.0.0/20"
}

variable "pods_cidr" {
  description = "Secondary range for GKE pod IPs (VPC-native cluster)"
  type        = string
  default     = "10.31.0.0/16"
}

variable "services_cidr" {
  description = "Secondary range for GKE service IPs (VPC-native cluster)"
  type        = string
  default     = "10.32.0.0/20"
}

variable "kubernetes_version" {
  description = "GKE release channel to subscribe the cluster to (REGULAR tracks a recent stable version automatically)"
  type        = string
  default     = "REGULAR"
}

variable "node_machine_type" {
  type    = string
  default = "e2-medium"
}

variable "node_desired_size" {
  type    = number
  default = 3
}

variable "db_tier" {
  description = "Cloud SQL machine tier"
  type        = string
  default     = "db-custom-1-3840"
}

variable "db_password" {
  description = "Postgres master password. Pass via TF_VAR_db_password or a secrets backend — never commit a real value."
  type        = string
  sensitive   = true
}

variable "redis_tier" {
  description = "Memorystore service tier (BASIC or STANDARD_HA)"
  type        = string
  default     = "BASIC"
}

variable "redis_memory_size_gb" {
  type    = number
  default = 1
}

variable "enable_cdn" {
  description = "Provision a global external HTTPS load balancer with Cloud CDN in front of the ingress-nginx LoadBalancer Service. Requires cdn_origin_ip — leave false until that IP exists (see main.tf comment on the CDN resources)."
  type        = bool
  default     = false
}

variable "cdn_origin_ip" {
  description = "External IP of the ingress-nginx LoadBalancer Service to use as the Cloud CDN origin. Only required when enable_cdn = true."
  type        = string
  default     = ""
}

variable "cdn_origin_port" {
  description = "Port the ingress-nginx LoadBalancer Service listens on. Only used when enable_cdn = true."
  type        = number
  default     = 80
}
