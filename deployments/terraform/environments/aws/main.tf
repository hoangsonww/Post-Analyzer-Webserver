terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Remote state is deliberately not configured here — this repo is
  # local-first. Before any real `terraform apply`, add an `s3` backend
  # block (with a DynamoDB lock table) pointing at infra dedicated to
  # this project, then `terraform init -migrate-state`.
}

provider "aws" {
  region = var.region
}

module "post_analyzer" {
  source = "../../modules/aws"

  project                = var.project
  environment            = var.environment
  region                 = var.region
  db_password            = var.db_password
  enable_cdn             = var.enable_cdn
  cdn_origin_domain_name = var.cdn_origin_domain_name
}
