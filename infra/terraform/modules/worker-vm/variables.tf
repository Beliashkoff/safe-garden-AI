variable "provider_kind" {
  description = "Куда выкатываем worker-VM: hostkey (default, prod) | hetzner | ovh. Параметризовано для DR-переезда — см. ARCHITECTURE.md §11.7."
  type        = string
  default     = "hostkey"
  validation {
    condition     = contains(["hostkey", "hetzner", "ovh"], var.provider_kind)
    error_message = "provider_kind должен быть одним из: hostkey, hetzner, ovh."
  }
}

variable "name" {
  description = "Имя VM (используется в Hetzner/OVH, для HostKey — справочно)."
  type        = string
  default     = "safegarden-worker"
}

variable "ssh_key_id" {
  description = "ID SSH-ключа в Hetzner Cloud (для Hetzner). На HostKey/OVH — public key передаётся другим путём."
  type        = string
  default     = ""
}

variable "ssh_public_key" {
  description = "Публичный SSH-ключ в формате OpenSSH. Прокидывается в cloud-init."
  type        = string
  default     = ""
}

variable "allowed_source_ips" {
  description = "Список IP/CIDR, которым разрешён доступ на 443 (mTLS endpoint). В prod — единственный IP бэкенд-VM в Yandex Cloud."
  type        = list(string)
  default     = []
}

# === Hetzner ===
variable "hetzner_server_type" {
  description = "Тип сервера Hetzner Cloud (например cx22, cpx21)."
  type        = string
  default     = "cx22"
}

variable "hetzner_location" {
  description = "Локация Hetzner: fsn1 (Falkenstein), nbg1 (Nuremberg), hel1 (Helsinki). Frankfurt отсутствует — fsn1 ближайший."
  type        = string
  default     = "fsn1"
}

# === OVH ===
variable "ovh_service_name" {
  description = "OVH Public Cloud project ID."
  type        = string
  default     = ""
}

variable "ovh_region" {
  description = "Регион OVH Public Cloud."
  type        = string
  default     = "DE1"
}

variable "ovh_flavor_name" {
  description = "Тип инстанса OVH (например s1-2, b2-7)."
  type        = string
  default     = "s1-2"
}

variable "ovh_image_name" {
  description = "Имя образа OVH (Ubuntu 24.04)."
  type        = string
  default     = "Ubuntu 24.04"
}

# === HostKey (manual provisioning) ===
variable "hostkey_manual_ip" {
  description = "Публичный IP HostKey-VM. Заполняется руками после провижининга (см. modules/worker-vm/README.md)."
  type        = string
  default     = ""
}

variable "hostkey_region" {
  description = "Локация HostKey (для метаданных): Frankfurt 1 / Amsterdam EuNetworks."
  type        = string
  default     = "Frankfurt 1"
}
