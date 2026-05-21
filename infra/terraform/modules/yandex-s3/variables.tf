variable "bucket" {
  description = "Имя бакета (глобально уникальное)."
  type        = string
}

variable "folder_id" {
  description = "ID каталога Yandex Cloud."
  type        = string
}

variable "default_storage_class" {
  description = "STANDARD | COLD | ICE."
  type        = string
  default     = "STANDARD"
}

variable "max_size_bytes" {
  description = "Лимит размера бакета в байтах (0 — без лимита)."
  type        = number
  default     = 0
}

variable "lifecycle_expiration_days" {
  description = "Через сколько дней удалять объекты по правилу cleanup. 0 — правило не создаётся."
  type        = number
  default     = 0
}
