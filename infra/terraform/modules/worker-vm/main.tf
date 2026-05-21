terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.48"
    }
    ovh = {
      source  = "ovh/ovh"
      version = "~> 0.51"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

locals {
  is_hostkey = var.provider_kind == "hostkey"
  is_hetzner = var.provider_kind == "hetzner"
  is_ovh     = var.provider_kind == "ovh"

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

# --- OVH Public Cloud (DR target) ---

resource "ovh_cloud_project_instance" "worker" {
  count        = local.is_ovh ? 1 : 0
  service_name = var.ovh_service_name
  region       = var.ovh_region

  billing_period = "hourly"

  boot_from {
    image_name = var.ovh_image_name
  }

  flavor {
    flavor_name = var.ovh_flavor_name
  }

  network {
    public = true
  }

  user_data = local.cloud_init
}

# --- HostKey (manual) ---
# У HostKey нет нативного Terraform-провайдера. VM создаётся в личном
# кабинете руками, IP вписывается в hostkey_manual_ip. Этот ресурс
# хранит фиксацию параметров в state — чтобы terraform plan показывал
# дрейф, если кто-то поменяет значение.

resource "null_resource" "hostkey_manual" {
  count = local.is_hostkey ? 1 : 0
  triggers = {
    name   = var.name
    region = var.hostkey_region
    ip     = var.hostkey_manual_ip
    note   = "HostKey VM provisioned manually; see modules/worker-vm/README.md"
  }
}
