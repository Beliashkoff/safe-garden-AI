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

  # Remote state в Yandex Object Storage (S3-совместимо). Включается при первом
  # apply (Этап 2.2 runbook, см. envs/prod/README.md): сначала создаётся приватный
  # бакет `safe-garden-tfstate`, затем раскомментировать блок и `terraform init
  # -migrate-state`. До этого — local state (gitignored). Оставлено
  # закомментированным, чтобы CI `init -backend=false` + `validate` проходили.
  #
  # backend "s3" {
  #   endpoints = { s3 = "https://storage.yandexcloud.net" }
  #   bucket   = "safe-garden-tfstate"
  #   key      = "prod/terraform.tfstate"
  #   region   = "ru-central1"
  #
  #   skip_region_validation      = true
  #   skip_credentials_validation = true
  #   skip_requesting_account_id  = true
  #   skip_s3_checksum            = true
  #   # AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY = static key сервис-аккаунта YC.
  # }
}
