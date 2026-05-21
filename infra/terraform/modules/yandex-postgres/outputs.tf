output "id" {
  description = "ID кластера."
  value       = yandex_mdb_postgresql_cluster.this.id
}

output "fqdn" {
  description = "FQDN основного хоста."
  value       = yandex_mdb_postgresql_cluster.this.host[0].fqdn
}
