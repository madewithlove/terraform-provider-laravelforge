resource "laravelforge_recipes" "deploy_tooling" {
  organization = "my-org"
  name         = "Install build tooling"
  user         = "forge"
  script       = <<-EOT
    sudo apt-get update
    sudo apt-get install -y jq
  EOT
}
