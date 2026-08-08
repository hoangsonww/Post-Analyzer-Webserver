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
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "vpc_cidr" {
  type    = string
  default = "10.20.0.0/16"
}

variable "availability_zones" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b"]
}

variable "kubernetes_version" {
  type    = string
  default = "1.30"
}

variable "node_instance_type" {
  type    = string
  default = "t3.medium"
}

variable "node_desired_size" {
  type    = number
  default = 3
}

variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_password" {
  description = "Postgres master password. Pass via TF_VAR_db_password or a secrets backend — never commit a real value."
  type        = string
  sensitive   = true
}

variable "redis_node_type" {
  type    = string
  default = "cache.t4g.micro"
}
