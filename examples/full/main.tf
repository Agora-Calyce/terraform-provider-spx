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

# Exemple complet : une VM Debian servant une page nginx sur le port 80,
# exposée sur Internet par une EIP (HTTP + SSH) et dans le VPC par un load balancer,
# et protégée par un groupe de sécurité.
locals {
  # IP fixe de la VM : cible du DNAT de l'EIP
  web_ip = "10.11.12.10"
  # Label porté par la VM : ciblé par le load balancer et le groupe de sécurité
  web_label = "user.superphenix.net/app:web"
}

##########################################################################
# Gestion du réseau
resource "spx_vpc" "example" {
  name = "mon-vpc-terraform"
}
output "vpc_id" {
  value = spx_vpc.example.id
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


# Routes statiques du VPC (exemple : réseau distant joignable via une passerelle VPN en 10.11.12.254)
resource "spx_vpc_static_routes" "example" {
  vpc_id = spx_vpc.example.id

  route {
    cidr        = "192.168.100.0/24"
    next_hop_ip = "10.11.12.254"
    route_table = spx_subnet.example.id
  }
}


# EIP publique : HTTP et SSH redirigés vers la VM
resource "spx_eip" "example_snat_dnat" {
  name      = "mon-eip-terraform-snat-dnat"
  subnet_id = spx_subnet.example.id
  snat      = [spx_subnet.example.ipv4]
  dnat {
    external_port = "80"
    internal_ip   = local.web_ip
    internal_port = "80"
    protocol      = "tcp"
  }
  dnat {
    external_port = "22"
    internal_ip   = local.web_ip
    internal_port = "22"
    protocol      = "tcp"
  }
}
output "eip_snat_dnat_id" {
  value = spx_eip.example_snat_dnat.id
}
output "eip_snat_dnat_external_ip" {
  value = spx_eip.example_snat_dnat.external_ip
}

resource "spx_eip" "example_internal" {
  name        = "mon-eip-terraform-internal"
  subnet_id   = spx_subnet.example.id
  internal_ip = "10.11.12.11"
}
output "eip_internal_id" {
  value = spx_eip.example_internal.id
}
output "eip_internal_external_ip" {
  value = spx_eip.example_internal.external_ip
}


##########################################################################
# Gestion des disks
data "spx_storage_classes" "all" {}

output "storage_classes" {
  value = data.spx_storage_classes.all.names
}

# Disque système créé à partir de l'image cloud Debian 13
resource "spx_disk" "boot" {
  name          = "mon-disque-terraform-boot"
  size          = 20
  storage_class = "default" # valeurs possibles : output storage_classes
  source {
    type = "http"
    url  = "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2"
  }
}
output "boot_disk_id" {
  value = spx_disk.boot.id
}

# Disque de données vierge
resource "spx_disk" "example" {
  name          = "mon-disque-terraform"
  size          = 2
  storage_class = "default"
  source {
    type = "blank"
  }
}
output "disk_id" {
  value = spx_disk.example.id
}


##########################################################################
# Gestion des clés SSH
resource "spx_ssh_key" "example" {
  name       = "ma-cle-ssh-terraform"
  public_key = trimspace(file(pathexpand(var.ssh_public_key_path)))
}
output "ssh_key_fingerprint" {
  value = spx_ssh_key.example.fingerprint
}


##########################################################################
# Gestion des load-balancers
# VIP interne au VPC répartissant le HTTP sur les VM portant le label web
resource "spx_load_balancer" "example_selectors" {
  name = "mon-lb-terraform-selectors"
  vip  = "198.18.0.20"

  selectors = [local.web_label]

  ports {
    port        = 80
    target_port = 80
    protocol    = "TCP"
  }
}
output "lb_selectors_id" {
  value = spx_load_balancer.example_selectors.id
}


##########################################################################
# Gestion des instances
data "spx_vm_types" "all" {}

output "vm_types" {
  value = data.spx_vm_types.all.types
}

resource "spx_instance" "example" {
  name         = "ma-vm-terraform"
  run_strategy = "Always"
  vm_type      = "debian" # valeurs possibles : output vm_types
  cpu          = 2
  memory       = 4

  network {
    order     = 0
    subnet_id = spx_subnet.example.id
    ipv4      = local.web_ip
  }

  disks {
    order   = 0
    disk_id = spx_disk.boot.id
  }
  disks {
    order   = 1
    disk_id = spx_disk.example.id
  }

  # Clé injectée par SPX dans le compte par défaut de l'image (debian)
  ssh_keys = [spx_ssh_key.example.id]

  # Configuration cloud-init personnalisée : remplace celle générée par SPX
  cloud_init {
    custom = true
    config = <<-EOT
      #cloud-config
      package_update: true
      packages:
        - nginx
        - openssh-server

      # Connexion SSH par clé uniquement, sur le compte par défaut "debian"
      ssh_pwauth: false
      disable_root: true
      ssh_authorized_keys:
        - ${trimspace(file(pathexpand(var.ssh_public_key_path)))}

      write_files:
        - path: /etc/systemd/resolved.conf.d/dns_servers.conf
          content: |
            [Resolve]
            DNS=1.1.1.1 1.0.0.1
          permissions: '0644'
        - path: /var/www/html/index.html
          content: |
            <h1>Hello from ma-vm-terraform</h1>
          permissions: '0644'

      runcmd:
        - systemctl restart systemd-resolved
        - systemctl enable --now ssh
        - systemctl enable --now nginx
    EOT
  }

  labels = [local.web_label]
}
output "instance_id" {
  value = spx_instance.example.id
}
output "instance_ip" {
  value = spx_instance.example.ip_address
}
output "ssh_command" {
  value = "ssh debian@${spx_eip.example_snat_dnat.external_ip}"
}
output "http_url" {
  value = "http://${spx_eip.example_snat_dnat.external_ip}"
}


##########################################################################
# Gestion des groupes de sécurité
# Protège les VM portant le label web du subnet : HTTP ouvert à tous (EIP et load balancer),
# SSH limité à var.admin_cidr, trafic sortant autorisé (DNS, paquets...).
resource "spx_security_group" "web" {
  name        = "mon-sg-terraform-web"
  description = "HTTP public, SSH depuis admin_cidr"
  subnet_ids  = [spx_subnet.example.id]

  target {
    match_labels = [local.web_label]
  }

  ingress {
    ports {
      port     = 80
      protocol = "TCP"
    }
    from {
      ip_block {
        cidr = "0.0.0.0/0"
      }
    }
  }

  ingress {
    ports {
      port     = 22
      protocol = "TCP"
    }
    from {
      ip_block {
        cidr = var.admin_cidr
      }
    }
  }

  egress {
    allow_all = true
  }
}
output "security_group_web_id" {
  value = spx_security_group.web.id
}
