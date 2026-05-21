variable "name" {
  description = "Имя инстанса."
  type        = string
}

variable "zone" {
  description = "Зона Yandex Cloud (ru-central1-a/b/d)."
  type        = string
  default     = "ru-central1-a"
}

variable "subnet_id" {
  description = "ID подсети VPC."
  type        = string
}

variable "security_group_ids" {
  description = "Список ID security group."
  type        = list(string)
  default     = []
}

variable "platform_id" {
  description = "Платформа: standard-v3 (Ice Lake)."
  type        = string
  default     = "standard-v3"
}

variable "cores" {
  description = "Количество vCPU."
  type        = number
  default     = 2
}

variable "memory" {
  description = "Размер RAM в ГБ."
  type        = number
  default     = 4
}

variable "disk_size" {
  description = "Размер boot-диска в ГБ."
  type        = number
  default     = 30
}

variable "image_family" {
  description = "Семейство образа (ubuntu-2404-lts)."
  type        = string
  default     = "ubuntu-2404-lts"
}

variable "service_account_id" {
  description = "Service account для VM (доступ к Lockbox, Container Registry)."
  type        = string
}

variable "ssh_public_key" {
  description = "Публичный SSH-ключ в формате OpenSSH."
  type        = string
}

variable "user_data" {
  description = "Cloud-init payload (опционально, перекрывает встроенный)."
  type        = string
  default     = ""
}
