output "api_public_ip" {
  description = "Публичный IP бэкенда (для A-записи api.agronomai.site)."
  value       = module.api_vm.public_ip
}

output "worker_ip" {
  description = "Публичный IP worker-VM."
  value       = module.worker_vm.ip
}

output "worker_provider" {
  description = "Какой провайдер использован для worker-VM."
  value       = module.worker_vm.provider_kind
}

output "postgres_fqdn" {
  description = "FQDN мастера PostgreSQL."
  value       = module.postgres.fqdn
}

output "redis_id" {
  description = "ID Redis-кластера (FQDN недоступен напрямую — через CLI)."
  value       = module.redis.id
}

output "media_bucket" {
  description = "Имя бакета для медиа."
  value       = module.media_bucket.bucket
}
