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
  description = "Un groupe de sécurité d'exemple avec labels"

  # Facultatif : limite le groupe de sécurité à certains subnets (10 maximum).
  # Si non spécifié, il s'applique à tous les subnets du projet.
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
      port = 443 # protocole TCP par défaut
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
  description = "Un groupe de sécurité d'exemple avec expression"

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
