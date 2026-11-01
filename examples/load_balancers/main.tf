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

resource "spx_load_balancer" "example_selectors" {
  name = "mon-lb-terraform-selectors"
  vip  = "198.18.0.20"

  selectors = ["app:frontend"]

  ports {
    port        = 80
    target_port = 8080
    protocol    = "TCP"
  }
}

resource "spx_load_balancer" "example_endpoints" {
  name = "mon-lb-terraform-endpoints"
  vip  = "198.18.0.21"

  endpoints = ["10.11.12.10", "10.11.12.11"]

  ports {
    port        = 80
    target_port = 8080
    protocol    = "TCP"
  }
  ports {
    port        = 81
    target_port = 8081
    protocol    = "TCP"
  }
}

output "lb_selectors_id" {
  value = spx_load_balancer.example_selectors.id
}
output "lb_endpoints_id" {
  value = spx_load_balancer.example_endpoints.id
}

