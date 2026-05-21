terraform {
  required_version = ">= 1.6.0"

  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
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

  # backend будет добавлен в Этапе 2.2 — remote state в Yandex Object Storage.
  # До этого момента state локальный (gitignored).
}
