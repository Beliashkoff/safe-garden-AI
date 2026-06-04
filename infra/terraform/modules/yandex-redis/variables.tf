variable "name" {
  description = "Имя кластера."
  type        = string
}

variable "environment" {
  description = "PRESTABLE | PRODUCTION."
  type        = string
  default     = "PRODUCTION"
}

variable "network_id" {
  description = "VPC network ID."
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID хоста."
  type        = string
}

variable "zone" {
  description = "Зона хоста."
  type        = string
  default     = "ru-central1-a"
}

variable "redis_version" {
  description = "Версия Redis/Valkey. YC принимает только *-valkey: 7.2-valkey, 8.0-valkey, 8.1-valkey, 9.0-valkey, 9.1-valkey."
  type        = string
  default     = "7.2-valkey"
}

variable "password" {
  description = "Пароль Redis (AUTH). Генерируется в env, хранится в Lockbox."
  type        = string
  sensitive   = true
}

variable "resource_preset_id" {
  description = "Размер хоста Redis/Valkey (current-gen, доступен в zone d). b3-c1-m4 = 2 vCPU / 4 GB, burstable — хватает для rate-limit (fail-open)."
  type        = string
  default     = "b3-c1-m4"
}

variable "disk_size" {
  description = "Размер диска в ГБ."
  type        = number
  default     = 16
}

variable "sharded" {
  description = "Шардирование (false для старта)."
  type        = bool
  default     = false
}

variable "security_group_ids" {
  description = "Список ID security group."
  type        = list(string)
  default     = []
}

variable "tls_enabled" {
  description = "Включить TLS-порт (6380). false → обычный порт 6379, совпадает с SG."
  type        = bool
  default     = false
}
