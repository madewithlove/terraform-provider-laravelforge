resource "laravelforge_servers" "example" {
  cloud_provider = "ocean2"
  name           = "example-name"
  organization   = "my-org"
  type           = "app"
  ubuntu_version = "22.04"
}
