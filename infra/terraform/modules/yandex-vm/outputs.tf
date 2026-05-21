output "id" {
  description = "ID инстанса."
  value       = yandex_compute_instance.this.id
}

output "public_ip" {
  description = "Публичный IPv4."
  value       = yandex_compute_instance.this.network_interface[0].nat_ip_address
}

output "internal_ip" {
  description = "Внутренний IPv4."
  value       = yandex_compute_instance.this.network_interface[0].ip_address
}
