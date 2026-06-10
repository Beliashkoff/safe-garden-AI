provider "yandex" {
  token     = var.yc_token
  cloud_id  = var.yc_cloud_id
  folder_id = var.yc_folder_id
  zone      = var.yc_zone
}

provider "hcloud" {
  # Hetzner — только DR (worker_provider=hetzner). При hostkey ресурсы count=0
  # и токен не нужен, но провайдер всё равно конфигурируется и отвергает пустой
  # токен. Подставляем валидный по длине (64 символа) плейсхолдер, когда токен
  # не задан — API-вызовов к Hetzner при этом не происходит.
  token = var.hcloud_token != "" ? var.hcloud_token : join("", [for _ in range(64) : "0"])
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
  # Internal mTLS listener (:8090) для обратного вызова worker->backend
  # recommend_fertilizer (ARCH §11.3). Открыт ТОЛЬКО для публичного IP worker-VM;
  # аутентификация — клиентский серт (worker-client). Пропускается, если
  # worker_manual_ip не задан.
  dynamic "ingress" {
    for_each = var.worker_manual_ip != "" ? [1] : []
    content {
      protocol       = "TCP"
      port           = 8090
      v4_cidr_blocks = ["${var.worker_manual_ip}/32"]
    }
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
# SA создан вручную (на нём статический ключ для tfstate-backend и роли),
# поэтому ссылаемся на него как на внешний ресурс, а не создаём заново —
# иначе конфликт имени, а на destroy потерялся бы ключ доступа к state.

data "yandex_iam_service_account" "api" {
  name = "safegarden-api"
}

# --- Compute (api) ---

module "api_vm" {
  source = "../../modules/yandex-vm"

  name               = "safegarden-api"
  zone               = var.yc_zone
  subnet_id          = yandex_vpc_subnet.main.id
  security_group_ids = [yandex_vpc_security_group.api.id]
  service_account_id = data.yandex_iam_service_account.api.id
  ssh_public_key     = var.ssh_public_key
}

# --- Managed PostgreSQL ---

resource "random_password" "pg" {
  length  = 24
  special = false
}

module "postgres" {
  source = "../../modules/yandex-postgres"

  name               = "safegarden-pg"
  network_id         = yandex_vpc_network.main.id
  subnet_id          = yandex_vpc_subnet.main.id
  zone               = var.yc_zone
  security_group_ids = [yandex_vpc_security_group.db.id]
  db_password        = random_password.pg.result
}

# --- Managed Redis ---

resource "random_password" "redis" {
  length  = 24
  special = false
}

module "redis" {
  source = "../../modules/yandex-redis"

  name               = "safegarden-redis"
  network_id         = yandex_vpc_network.main.id
  zone               = var.yc_zone
  subnet_id          = yandex_vpc_subnet.main.id
  security_group_ids = [yandex_vpc_security_group.db.id]
  password           = random_password.redis.result
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

  provider_kind  = var.worker_provider
  name           = "safegarden-worker"
  ssh_public_key = var.ssh_public_key
  ssh_key_id     = var.hcloud_ssh_key_id
  manual_ip      = var.worker_manual_ip

  # IP бэкенда — единственный источник, которому worker открывает 443 (mTLS).
  # Для manual/hostkey firewall настраивается на самой VM (UFW); для hetzner —
  # уходит в hcloud_firewall.
  allowed_source_ips = compact([module.api_vm.public_ip])
}
