resource "laravelforge_site_domains" "example" {
  allow_wildcard_subdomains = false
  name                      = "laravel.com"
  organization              = "my-org"
  server                    = 123456
  site                      = 123456
  www_redirect_type         = "from-www"
}
