output "ip" {
  description = "Публичный IPv4 worker-VM. Используется бэкендом как mTLS endpoint."
  value = try(
    hcloud_server.worker[0].ipv4_address,
    ovh_cloud_project_instance.worker[0].addresses[0].ip,
    var.hostkey_manual_ip,
  )
}

output "provider_kind" {
  description = "Какой провайдер фактически инстанциирован."
  value       = var.provider_kind
}

output "name" {
  description = "Имя VM."
  value       = var.name
}
