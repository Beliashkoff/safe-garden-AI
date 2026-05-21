terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }
}

resource "yandex_storage_bucket" "this" {
  bucket                = var.bucket
  folder_id             = var.folder_id
  default_storage_class = var.default_storage_class

  max_size = var.max_size_bytes

  anonymous_access_flags {
    read = false
    list = false
  }

  dynamic "lifecycle_rule" {
    for_each = var.lifecycle_expiration_days > 0 ? [1] : []
    content {
      id      = "cleanup-old"
      enabled = true
      expiration {
        days = var.lifecycle_expiration_days
      }
    }
  }
}
