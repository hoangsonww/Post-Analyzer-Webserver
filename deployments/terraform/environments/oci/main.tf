terraform {
  required_version = ">= 1.5"
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 5.0"
    }
  }
  # Remote state intentionally not configured — see environments/aws/main.tf.
}

module "post_analyzer" {
  source = "../../modules/oci"

  project           = var.project
  environment       = var.environment
  compartment_id    = var.compartment_id
  region            = var.region
  db_admin_password = var.db_admin_password
}
