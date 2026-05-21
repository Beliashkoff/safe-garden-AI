terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

resource "yandex_mdb_postgresql_cluster" "this" {
  name        = var.name
  environment = var.environment
  network_id  = var.network_id

  security_group_ids = var.security_group_ids

  config {
    version = var.pg_version
    resources {
      resource_preset_id = var.resource_preset_id
      disk_type_id       = var.disk_type_id
      disk_size          = var.disk_size
    }

    backup_window_start {
      hours   = 2
      minutes = 0
    }
  }

  host {
    zone      = var.zone
    subnet_id = var.subnet_id
  }
}
