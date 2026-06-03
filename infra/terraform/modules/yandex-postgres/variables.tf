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

variable "resource_preset_id" {
  description = "Размер хоста: s2.micro (2 vCPU / 8 GB) на старт."
  type        = string
  default     = "s2.micro"
}

variable "disk_size" {
  description = "Размер диска в ГБ."
  type        = number
  default     = 20
}

variable "disk_type_id" {
  description = "network-ssd | network-ssd-nonreplicated | network-hdd."
  type        = string
  default     = "network-ssd"
}

variable "pg_version" {
  description = "Версия PostgreSQL."
  type        = string
  default     = "16"
}

variable "db_user" {
  description = "Имя пользователя приложения."
  type        = string
  default     = "safegarden"
}

variable "db_name" {
  description = "Имя базы приложения."
  type        = string
  default     = "safegarden"
}

variable "db_password" {
  description = "Пароль пользователя приложения. Генерируется в env, хранится в Lockbox."
  type        = string
  sensitive   = true
}

variable "security_group_ids" {
  description = "Список ID security group для кластера."
  type        = list(string)
  default     = []
}
