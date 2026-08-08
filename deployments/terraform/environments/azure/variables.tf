variable "project" {
  type    = string
  default = "post-analyzer"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "location" {
  type    = string
  default = "eastus"
}

variable "db_admin_password" {
  type      = string
  sensitive = true
}

variable "enable_cdn" {
  description = "Provision an Azure Front Door profile. Requires cdn_origin_hostname — see modules/azure/main.tf."
  type        = bool
  default     = false
}

variable "cdn_origin_hostname" {
  description = "AKS ingress-nginx LoadBalancer hostname, known only after the K8s ingress is deployed to this cluster. Only required when enable_cdn = true."
  type        = string
  default     = ""
}
