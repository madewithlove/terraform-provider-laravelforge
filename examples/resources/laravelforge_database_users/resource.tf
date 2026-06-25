resource "laravelforge_database_users" "app" {
  organization = "my-org"
  server       = 123456
  name         = "app_user"
  password     = var.database_user_password
  read_only    = false
  database_ids = [laravelforge_database_schemas.app.database]
}

variable "database_user_password" {
  type      = string
  sensitive = true
}
