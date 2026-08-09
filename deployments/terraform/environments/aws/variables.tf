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
  default = "us-east-1"
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "enable_cdn" {
  description = "Provision a CloudFront distribution. Requires cdn_origin_domain_name — see modules/aws/main.tf."
  type        = bool
  default     = false
}

variable "cdn_origin_domain_name" {
  description = "ingress-nginx ELB hostname, known only after the K8s ingress is deployed to this cluster. Only required when enable_cdn = true."
  type        = string
  default     = ""
}
