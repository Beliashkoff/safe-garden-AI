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

  # Защита от случайного удаления прод-кластера (terraform destroy / yc delete
  # потребуют сначала снять флаг). Бэкапы — отдельный слой (ежедневно, 7д, PITR).
  deletion_protection = true

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

resource "yandex_mdb_postgresql_user" "app" {
  cluster_id = yandex_mdb_postgresql_cluster.this.id
  name       = var.db_user
  password   = var.db_password
}

resource "yandex_mdb_postgresql_database" "app" {
  cluster_id = yandex_mdb_postgresql_cluster.this.id
  name       = var.db_name
  owner      = yandex_mdb_postgresql_user.app.name

  # YC Managed PostgreSQL forbids CREATE EXTENSION by app users; extensions are
  # enabled at the cluster level. Migration 0001_extensions.sql needs these.
  extension {
    name = "citext"
  }
  extension {
    name = "pgcrypto"
  }
}
