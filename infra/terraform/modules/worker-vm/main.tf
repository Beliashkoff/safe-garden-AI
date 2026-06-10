terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.48"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

locals {
  # "hostkey" остаётся legacy-алиасом ручного провижининга.
  is_manual  = var.provider_kind == "manual" || var.provider_kind == "hostkey"
  is_hetzner = var.provider_kind == "hetzner"

  cloud_init = templatefile("${path.module}/cloud-init.yaml.tftpl", {
    ssh_public_key     = var.ssh_public_key
    allowed_source_ips = var.allowed_source_ips
  })
}

# --- Hetzner Cloud (DR target) ---

resource "hcloud_firewall" "worker" {
  count = local.is_hetzner ? 1 : 0
  name  = "${var.name}-fw"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = var.allowed_source_ips
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = var.allowed_source_ips
  }
}

resource "hcloud_server" "worker" {
  count        = local.is_hetzner ? 1 : 0
  name         = var.name
  server_type  = var.hetzner_server_type
  image        = "ubuntu-24.04"
  location     = var.hetzner_location
  ssh_keys     = var.ssh_key_id == "" ? [] : [var.ssh_key_id]
  user_data    = local.cloud_init
  firewall_ids = [hcloud_firewall.worker[0].id]

  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
}

# --- Manual provisioning (VPS вне Yandex) ---
# У ручного VPS нет нативного Terraform-провайдера. VM создаётся в панели
# провайдера руками, IP вписывается в manual_ip. Этот ресурс хранит фиксацию
# параметров в state — чтобы terraform plan показывал дрейф при их изменении.

resource "null_resource" "manual" {
  count = local.is_manual ? 1 : 0
  triggers = {
    name   = var.name
    region = var.manual_region
    ip     = var.manual_ip
    note   = "Worker VM provisioned manually (VPS outside RU); see modules/worker-vm/README.md"
  }
}

moved {
  from = null_resource.hostkey_manual
  to   = null_resource.manual
}
