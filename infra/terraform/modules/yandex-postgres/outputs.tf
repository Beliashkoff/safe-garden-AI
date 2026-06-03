output "id" {
  description = "ID кластера."
  value       = yandex_mdb_postgresql_cluster.this.id
}

output "fqdn" {
  description = "FQDN основного хоста."
  value       = yandex_mdb_postgresql_cluster.this.host[0].fqdn
}

output "db_user" {
  description = "Пользователь приложения."
  value       = yandex_mdb_postgresql_user.app.name
}

output "db_name" {
  description = "База приложения."
  value       = yandex_mdb_postgresql_database.app.name
}
