resource "laravelforge_heartbeats" "example" {
  frequency    = 1
  grace_period = 1
  name         = "My Heartbeat"
  organization = "my-org"
  server       = 123456
  site         = 123456
}
