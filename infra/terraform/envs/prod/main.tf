provider "yandex" {
  token     = var.yc_token
  cloud_id  = var.yc_cloud_id
  folder_id = var.yc_folder_id
  zone      = var.yc_zone
}

provider "hcloud" {
  token = var.hcloud_token
}

provider "ovh" {
  endpoint           = var.ovh_endpoint
  application_key    = var.ovh_application_key
  application_secret = var.ovh_application_secret
  consumer_key       = var.ovh_consumer_key
}

# --- VPC ---

resource "yandex_vpc_network" "main" {
  name = "safegarden-main"
}

resource "yandex_vpc_subnet" "main" {
  name           = "safegarden-main-${var.yc_zone}"
  zone           = var.yc_zone
  network_id     = yandex_vpc_network.main.id
  v4_cidr_blocks = ["10.10.0.0/24"]
}

resource "yandex_vpc_security_group" "api" {
  name       = "safegarden-api"
  network_id = yandex_vpc_network.main.id

  ingress {
    protocol       = "TCP"
    port           = 443
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    protocol       = "TCP"
    port           = 80
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    protocol       = "TCP"
    port           = 22
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    protocol       = "ANY"
    from_port      = 0
    to_port        = 65535
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "yandex_vpc_security_group" "db" {
  name       = "safegarden-db"
  network_id = yandex_vpc_network.main.id

  ingress {
    protocol          = "TCP"
    port              = 6432
    security_group_id = yandex_vpc_security_group.api.id
  }
  ingress {
    protocol          = "TCP"
    port              = 6379
    security_group_id = yandex_vpc_security_group.api.id
  }
}

# --- IAM ---

resource "yandex_iam_service_account" "api" {
  name = "safegarden-api"
}

# --- Compute (api) ---

module "api_vm" {
  source = "../../modules/yandex-vm"

  name               = "safegarden-api"
  zone               = var.yc_zone
  subnet_id          = yandex_vpc_subnet.main.id
  security_group_ids = [yandex_vpc_security_group.api.id]
  service_account_id = yandex_iam_service_account.api.id
  ssh_public_key     = var.ssh_public_key
}

# --- Managed PostgreSQL ---

module "postgres" {
  source = "../../modules/yandex-postgres"

  name               = "safegarden-pg"
  network_id         = yandex_vpc_network.main.id
  subnet_id          = yandex_vpc_subnet.main.id
  zone               = var.yc_zone
  security_group_ids = [yandex_vpc_security_group.db.id]
}

# --- Managed Redis ---

module "redis" {
  source = "../../modules/yandex-redis"

  name               = "safegarden-redis"
  network_id         = yandex_vpc_network.main.id
  security_group_ids = [yandex_vpc_security_group.db.id]
}

# --- Object Storage ---

module "media_bucket" {
  source = "../../modules/yandex-s3"

  bucket    = var.media_bucket_name
  folder_id = var.yc_folder_id
}

# --- LLM worker VM (вне Yandex Cloud, ARCH §11.2/11.7) ---

module "worker_vm" {
  source = "../../modules/worker-vm"

  provider_kind     = var.worker_provider
  name              = "safegarden-worker"
  ssh_public_key    = var.ssh_public_key
  ssh_key_id        = var.hcloud_ssh_key_id
  hostkey_manual_ip = var.hostkey_worker_ip
  ovh_service_name  = var.ovh_service_name

  # IP бэкенда станет известен после поднятия api_vm — добавим allowlist
  # отдельным шагом в 2.2 (terraform apply -target=module.worker_vm).
  allowed_source_ips = compact([module.api_vm.public_ip])
}
