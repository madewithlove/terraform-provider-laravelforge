resource "laravelforge_monitors" "example" {
  notify       = "taylor@laravel.com"
  operator     = "gte"
  organization = "my-org"
  server       = 123456
  threshold    = 90
  type         = "cpu_load"
}
