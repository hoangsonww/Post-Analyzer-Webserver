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

variable "kubernetes_version" {
  type    = string
  default = "1.30"
}

variable "node_vm_size" {
  type    = string
  default = "Standard_D2s_v5"
}

variable "node_count" {
  type    = number
  default = 3
}

variable "db_sku_name" {
  type    = string
  default = "B_Standard_B1ms"
}

variable "db_admin_password" {
  description = "Postgres admin password. Pass via TF_VAR_db_admin_password — never commit a real value."
  type        = string
  sensitive   = true
}

variable "redis_capacity" {
  type    = number
  default = 0 # smallest Basic tier size
}
