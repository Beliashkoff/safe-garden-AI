variable "provider_kind" {
  description = "Куда выкатываем worker-VM: manual (default, prod — ручной VPS вне РФ, IP в manual_ip) | hetzner (DR, Terraform поднимает сам) | hostkey (legacy-алиас ручного провижининга). См. ARCHITECTURE.md §11.7."
  type        = string
  default     = "manual"
  validation {
    condition     = contains(["manual", "hostkey", "hetzner"], var.provider_kind)
    error_message = "provider_kind должен быть одним из: manual, hostkey, hetzner."
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

# === Manual provisioning (VPS вне Yandex; default для prod) ===
variable "manual_ip" {
  description = "Публичный IP вручную провижененной worker-VM (VPS вне РФ). Заполняется после провижининга (см. modules/worker-vm/README.md)."
  type        = string
  default     = ""
}

variable "manual_region" {
  description = "Локация ручной worker-VM (для метаданных), напр. Finland."
  type        = string
  default     = "Finland"
}
