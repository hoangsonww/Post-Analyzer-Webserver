variable "project" {
  type    = string
  default = "post-analyzer"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "compartment_id" {
  description = "OCID of the compartment to create resources in"
  type        = string
}

variable "region" {
  type    = string
  default = "us-ashburn-1"
}

variable "kubernetes_version" {
  type    = string
  default = "v1.30.1"
}

variable "node_shape" {
  type    = string
  default = "VM.Standard.E4.Flex"
}

variable "node_pool_size" {
  type    = number
  default = 3
}

variable "db_admin_password" {
  description = "Postgres admin password. Pass via TF_VAR_db_admin_password — never commit a real value."
  type        = string
  sensitive   = true
}
