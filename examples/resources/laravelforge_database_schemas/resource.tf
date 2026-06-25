resource "laravelforge_database_schemas" "app" {
  organization = "my-org"
  server       = 123456
  name         = "app_production"
  user         = "app_user"
  password     = var.database_password
}

variable "database_password" {
  type      = string
  sensitive = true
}
