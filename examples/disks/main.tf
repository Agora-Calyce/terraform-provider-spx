terraform {
  required_providers {
    spx = {
      source  = "agora-calyce/spx"
      version = "1.0.0"
    }
  }
}

provider "spx" {
  endpoint   = var.endpoint
  api_token  = var.api_token
  az         = var.az
  orga_id    = var.orga_id
  project_id = var.project_id
}

data "spx_storage_classes" "all" {}

resource "spx_disk" "example" {
  name          = "mon-disque-terraform"
  size          = 2
  storage_class = "default" # valeurs possibles : output storage_classes
  labels        = ["user.superphenix.net/env:test"]
  source {
    type = "blank"
  }
}

output "disk_id" {
  value = spx_disk.example.id
}

output "storage_classes" {
  value = data.spx_storage_classes.all.names
}
