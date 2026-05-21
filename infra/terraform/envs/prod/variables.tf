variable "yc_token" {
  description = "OAuth-токен Yandex Cloud. В CI/локально — из YC_TOKEN env."
  type        = string
  sensitive   = true
  default     = ""
}

variable "yc_cloud_id" {
  description = "ID организации (cloud)."
  type        = string
  default     = ""
}

variable "yc_folder_id" {
  description = "ID каталога (folder) safe-garden-prod."
  type        = string
  default     = ""
}

variable "yc_zone" {
  description = "Зона по умолчанию."
  type        = string
  default     = "ru-central1-a"
}

variable "ssh_public_key" {
  description = "Публичный SSH-ключ оператора (одна строка в формате OpenSSH)."
  type        = string
  default     = ""
}

# === Worker провайдер ===
variable "worker_provider" {
  description = "Где живёт worker-VM: hostkey | hetzner | ovh. См. ARCH §11.7."
  type        = string
  default     = "hostkey"
}

variable "hcloud_token" {
  description = "API-токен Hetzner Cloud (нужен только для worker_provider=hetzner)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "hcloud_ssh_key_id" {
  description = "ID загруженного SSH-ключа в Hetzner Cloud."
  type        = string
  default     = ""
}

variable "ovh_endpoint" {
  description = "OVH API endpoint (ovh-eu / ovh-ca)."
  type        = string
  default     = "ovh-eu"
}

variable "ovh_application_key" {
  type      = string
  sensitive = true
  default   = ""
}
variable "ovh_application_secret" {
  type      = string
  sensitive = true
  default   = ""
}
variable "ovh_consumer_key" {
  type      = string
  sensitive = true
  default   = ""
}
variable "ovh_service_name" {
  description = "OVH Public Cloud project ID."
  type        = string
  default     = ""
}

variable "hostkey_worker_ip" {
  description = "Публичный IP HostKey-VM. Заполняется руками после провижининга."
  type        = string
  default     = ""
}

# === S3 ===
variable "media_bucket_name" {
  description = "Имя бакета для медиа пользователей."
  type        = string
  default     = "safe-garden-prod-media"
}
