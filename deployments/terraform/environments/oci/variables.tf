variable "project" {
  type    = string
  default = "post-analyzer"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "compartment_id" {
  type = string
}

variable "region" {
  type    = string
  default = "us-ashburn-1"
}

variable "db_admin_password" {
  type      = string
  sensitive = true
}
