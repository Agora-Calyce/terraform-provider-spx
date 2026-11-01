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

resource "spx_vpc" "example" {
  name = "mon-vpc-terraform"
}

resource "spx_subnet" "example" {
  name        = "mon-subnet-terraform"
  vpc_id      = spx_vpc.example.id
  protocol    = "IPv4"
  ipv4        = "10.11.12.0/24"
  private     = false
  dns_v4      = "8.8.8.8"
  nat_gateway = true
}

resource "spx_eip" "example_snat_dnat" {
  name      = "mon-eip-terraform-snat-dnat"
  subnet_id = spx_subnet.example.id
  snat      = [spx_subnet.example.ipv4]
  dnat {
    external_port = "80"
    internal_ip   = "10.11.12.10"
    internal_port = "80"
    protocol      = "tcp"
  }
}

resource "spx_eip" "example_internal" {
  name        = "mon-eip-terraform-internal"
  subnet_id   = spx_subnet.example.id
  internal_ip = "10.11.12.11"
}

output "eip_snat_dnat_id" {
  value = spx_eip.example_snat_dnat.id
}

output "eip_snat_dnat_external_ip" {
  value = spx_eip.example_snat_dnat.external_ip
}

output "eip_internal_id" {
  value = spx_eip.example_internal.id
}

output "eip_internal_external_ip" {
  value = spx_eip.example_internal.external_ip
}
