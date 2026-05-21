variable "name" {
  description = "Имя кластера."
  type        = string
}

variable "network_id" {
  description = "VPC network ID."
  type        = string
}

variable "resource_preset_id" {
  description = "Размер хоста: hm1.nano (1 vCPU / 6 GB) на старт."
  type        = string
  default     = "hm1.nano"
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
  description = "Включить TLS-порт."
  type        = bool
  default     = true
}
