resource "laravelforge_server_scheduled_jobs" "example" {
  command      = "echo $(whoami)"
  frequency    = "minutely"
  organization = "my-org"
  server       = 123456
  user         = "root"
}
