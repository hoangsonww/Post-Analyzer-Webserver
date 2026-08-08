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
