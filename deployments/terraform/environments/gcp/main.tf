terraform {
  required_version = ">= 1.5"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  # Remote state is deliberately not configured here — this repo is
  # local-first. Before any real `terraform apply`, add a `gcs` backend
  # block pointing at a bucket dedicated to this project's state, then
  # `terraform init -migrate-state`.
}

provider "google" {
  project = var.project_id
  region  = var.region
}

module "post_analyzer" {
  source = "../../modules/gcp"

  project_id      = var.project_id
  project         = var.project
  environment     = var.environment
  region          = var.region
  db_password     = var.db_password
  enable_cdn      = var.enable_cdn
  cdn_origin_ip   = var.cdn_origin_ip
  cdn_origin_port = var.cdn_origin_port
}
