output "bucket" {
  description = "Имя созданного бакета."
  value       = yandex_storage_bucket.this.bucket
}

output "id" {
  description = "ID бакета."
  value       = yandex_storage_bucket.this.id
}
