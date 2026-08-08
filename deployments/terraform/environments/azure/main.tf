terraform {
  required_version = ">= 1.5"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
  # Remote state intentionally not configured — see environments/aws/main.tf.
}

module "post_analyzer" {
  source = "../../modules/azure"

  project           = var.project
  environment       = var.environment
  location          = var.location
  db_admin_password = var.db_admin_password
}
