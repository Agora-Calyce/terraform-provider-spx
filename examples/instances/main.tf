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

resource "spx_vpc" "example" {
  name = "vpc-instance-test"
}

resource "spx_subnet" "example" {
  name     = "subnet-instance-test"
  vpc_id   = spx_vpc.example.id
  ipv4     = "10.10.20.0/24"
  protocol = "IPv4"
}

data "spx_storage_classes" "all" {}

output "storage_classes" {
  value = data.spx_storage_classes.all.names
}

resource "spx_disk" "boot" {
  name          = "disk-instance-boot"
  size          = 20
  storage_class = "default" # valeurs possibles : output storage_classes
  source {
    type = "http"
    url  = "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2"
  }
}

resource "spx_ssh_key" "admin" {
  name       = "ssh-instance-test"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC8n7+..."
}

data "spx_vm_types" "all" {}
resource "spx_instance" "example" {
  name         = "instance-test"
  run_strategy = "Always"
  vm_type      = "debian"
  cpu          = 2
  memory       = 4

  network {
    order     = 0
    subnet_id = spx_subnet.example.id
    model     = "virtio"
    # enabled     = false               # interface in "link down" state
    # mac_address = "02:00:00:aa:bb:cc" # static MAC address
  }

  disks {
    order   = 0
    bus     = "virtio"
    disk_id = spx_disk.boot.id
  }

  cloud_init {
    custom = false
    bus    = "virtio"
    config = "hostname: instance-test"
  }

  ssh_keys = [spx_ssh_key.admin.id]
  labels   = ["user.superphenix.net/env:test", "user.superphenix.net/app:web"]

  # Container disks (AZ catalog, e.g. drivers): if not set, the recommended set
  # is mounted on creation and existing mounts are kept afterwards; [] unmounts everything.
  # container_disks = ["virtio-win"]

  # Advanced options: any value not set is inherited from the vm_type.
  # advanced {
  #   tpm {
  #     enabled    = true
  #     persistent = true
  #   }
  #   efi {
  #     secure_boot = true
  #   }
  # }
}

output "instance_id" {
  value = spx_instance.example.id
}

output "instance_ip" {
  value = spx_instance.example.ip_address
}

output "instance_ready" {
  value = spx_instance.example.ready
}
