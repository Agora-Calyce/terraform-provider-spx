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

output "subnet_id" {
  value = spx_subnet.example.id
}

output "subnet_nat_gateway_ip" {
  value = spx_subnet.example.nat_gateway_ip
}
