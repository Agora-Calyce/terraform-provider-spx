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

resource "spx_ssh_key" "example" {
  name       = "ma-cle-ssh"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDfC9e9... user@host"
}

output "ssh_key_id" {
  value = spx_ssh_key.example.id
}

output "ssh_key_fingerprint" {
  value = spx_ssh_key.example.fingerprint
}
