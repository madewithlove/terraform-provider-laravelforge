resource "laravelforge_security_rules" "example" {
  credentials = [{
    username = "deploy"
    password = "secret"
  }]
  name         = "Restricted Access"
  organization = "my-org"
  server       = 123456
  site         = 123456
}
