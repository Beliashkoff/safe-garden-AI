terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

data "yandex_compute_image" "ubuntu" {
  family = var.image_family
}

locals {
  user_data = var.user_data != "" ? var.user_data : templatefile("${path.module}/cloud-init.yaml.tftpl", {
    ssh_public_key = var.ssh_public_key
  })
}

resource "yandex_compute_instance" "this" {
  name                      = var.name
  zone                      = var.zone
  platform_id               = var.platform_id
  service_account_id        = var.service_account_id
  allow_stopping_for_update = true

  resources {
    cores  = var.cores
    memory = var.memory
  }

  boot_disk {
    initialize_params {
      image_id = data.yandex_compute_image.ubuntu.id
      size     = var.disk_size
      type     = "network-ssd"
    }
  }

  network_interface {
    subnet_id          = var.subnet_id
    security_group_ids = var.security_group_ids
    nat                = true
  }

  metadata = {
    user-data = local.user_data
    ssh-keys  = "safegarden:${var.ssh_public_key}"
  }
}
