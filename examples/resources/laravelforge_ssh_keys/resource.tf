resource "laravelforge_ssh_keys" "deploy" {
  organization = "my-org"
  server       = 123456
  name         = "deploy-key"
  key          = file("~/.ssh/id_ed25519.pub")
  user         = "forge"
}
