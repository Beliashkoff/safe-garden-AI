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
  description = "Размер хоста: s2.micro (1 vCPU / 4 GB) на старт."
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

variable "security_group_ids" {
  description = "Список ID security group для кластера."
  type        = list(string)
  default     = []
}
