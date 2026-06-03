variable "provider_kind" {
  description = "Куда выкатываем worker-VM: hostkey (default, prod) | hetzner. Параметризовано для DR-переезда — см. ARCHITECTURE.md §11.7."
  type        = string
  default     = "hostkey"
  validation {
    condition     = contains(["hostkey", "hetzner"], var.provider_kind)
    error_message = "provider_kind должен быть одним из: hostkey, hetzner."
  }
}

variable "name" {
  description = "Имя VM (используется в Hetzner, для HostKey — справочно)."
  type        = string
  default     = "safegarden-worker"
}

variable "ssh_key_id" {
  description = "ID SSH-ключа в Hetzner Cloud (для Hetzner). На HostKey — public key передаётся другим путём."
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
