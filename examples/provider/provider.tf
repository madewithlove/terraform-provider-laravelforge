terraform {
  required_providers {
    laravelforge = {
      source = "madewithlove/laravelforge"
    }
  }
}

provider "laravelforge" {
  # The API token can also be supplied via the FORGE_API_TOKEN environment
  # variable, which is the recommended approach for secrets.
  api_token = var.forge_api_token

  # endpoint is optional and defaults to https://forge.laravel.com/api
  # endpoint = "https://forge.laravel.com/api"
}

variable "forge_api_token" {
  type      = string
  sensitive = true
}
