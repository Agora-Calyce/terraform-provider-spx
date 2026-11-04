terraform {
  required_providers {
    spx = {
      source = "registry.terraform.io/agora-calyce/spx"
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

resource "spx_security_group" "example" {
  name        = "example-security-group"
  description = "An example security group with labels"

  # Optional: limits the security group to specific subnets (10 maximum).
  # If not set, it applies to all subnets of the project.
  # subnet_ids = [spx_subnet.example.id]

  target {
    match_labels = ["user.superphenix.net/app:web"]
  }

  ingress {
    ports {
      port     = 80
      end_port = 81
      protocol = "TCP"
    }
    ports {
      port = 443 # TCP protocol by default
    }
    from {
      ip_block {
        cidr   = "0.0.0.0/0"
        except = ["10.0.0.0/8"]
      }
    }
  }

  # egress {
  #   ports {
  #     port     = 53
  #     protocol = "UDP"
  #   }
  #   to {
  #     ip_block {
  #       cidr = "8.8.8.8/32"
  #     }
  #   }
  # }
  # egress {
  #   allow_all = true
  # }
  # egress {
  #   deny_all = true
  # }
}

output "security_group_id" {
  value = spx_security_group.example.id
}

resource "spx_security_group" "example_expression" {
  name        = "example-security-group-expression"
  description = "An example security group with an expression"

  target {
    match_expressions {
      key      = "user.superphenix.net/env"
      operator = "In"
      values   = ["prod", "staging"]
    }
  }

  ingress {
    ports {
      port = 22
    }
    from {
      pod_selector {
        match_labels = ["user.superphenix.net/role:bastion"]
      }
    }
  }
}

output "security_group_expression_id" {
  value = spx_security_group.example_expression.id
}
