resource "laravelforge_site_scheduled_jobs" "example" {
  command      = "echo $(whoami)"
  frequency    = "minutely"
  organization = "my-org"
  server       = 123456
  site         = 123456
  user         = "root"
}
