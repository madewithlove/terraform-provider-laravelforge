resource "laravelforge_redirect_rules" "example" {
  from         = "/old-path"
  organization = "my-org"
  server       = 123456
  site         = 123456
  to           = "/new-path"
  type         = "redirect"
}
