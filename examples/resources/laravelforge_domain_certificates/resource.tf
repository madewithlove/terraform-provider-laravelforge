resource "laravelforge_domain_certificates" "example" {
  domain_record = 1
  organization  = "my-org"
  server        = 123456
  site          = 123456
  type          = "letsencrypt"
}
