resource "laravelforge_background_processes" "example" {
  command      = "php artisan custom:command"
  name         = "Custom command runner"
  organization = "my-org"
  processes    = 1
  server       = 123456
  user         = "forge"
}
