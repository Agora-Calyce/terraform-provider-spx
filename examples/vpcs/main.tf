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

output "vpc_id" {
  value = spx_vpc.example.id
}

resource "spx_subnet" "example" {
  name     = "mon-subnet-terraform"
  vpc_id   = spx_vpc.example.id
  protocol = "IPv4"
  ipv4     = "10.11.12.0/24"
}

# Routes statiques du VPC : ressource distincte de spx_vpc car chaque route référence
# un subnet du VPC (route_table). Gère l'ensemble des routes utilisateur du VPC ;
# les routes générées par les NAT Gateways ne sont pas concernées.
resource "spx_vpc_static_routes" "example" {
  vpc_id = spx_vpc.example.id

  route {
    cidr        = "192.168.100.0/24"
    next_hop_ip = "10.11.12.254"
    route_table = spx_subnet.example.id
  }

  route {
    policy      = "policySrc"
    cidr        = "10.11.12.128/25"
    next_hop_ip = "10.11.12.253"
    route_table = spx_subnet.example.id
  }
}
