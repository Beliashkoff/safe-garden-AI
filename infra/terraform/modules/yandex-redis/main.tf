terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

resource "yandex_mdb_redis_cluster_v2" "this" {
  name        = var.name
  network_id  = var.network_id
  sharded     = var.sharded
  tls_enabled = var.tls_enabled

  security_group_ids = var.security_group_ids

  resources {
    resource_preset_id = var.resource_preset_id
    disk_size          = var.disk_size
  }

  maintenance_window {
    type = "ANYTIME"
  }
}
