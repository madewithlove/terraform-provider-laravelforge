# Laravel Forge Terraform Provider

A Terraform provider for [Laravel Forge](https://forge.laravel.com), with
resource schemas generated from the official Forge OpenAPI document using the
[HashiCorp OpenAPI provider code generators](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator).

## Usage

```hcl
terraform {
  required_providers {
    laravelforge = {
      source = "madewithlove/laravelforge"
    }
  }
}

provider "laravelforge" {
  # api_token may also be set via the FORGE_API_TOKEN environment variable.
  api_token = var.forge_api_token
}

resource "laravelforge_recipes" "example" {
  organization = "my-org"
  name         = "Install jq"
  user         = "forge"
  script       = "sudo apt-get install -y jq"
}
```

### Provider configuration

| Argument    | Environment       | Description                                                        |
| ----------- | ----------------- | ------------------------------------------------------------------ |
| `api_token` | `FORGE_API_TOKEN` | Forge API token (required).                                        |
| `endpoint`  | `FORGE_ENDPOINT`  | API base URL. Defaults to `https://forge.laravel.com/api`.         |

## Implemented resources

CRUD (create, read, update where the API supports it, delete, and import) is
implemented for **all 28 resources**. The shared CRUD helpers live in
`internal/providerutil` and the HTTP client in `internal/forge`; each resource's
logic is in the hand-written `internal/provider/resource_*/*_resource.go` (the
`*_resource_gen.go` files are generated schemas and are not edited).

Live-validated end-to-end against the Forge API (create → read → update →
delete): `recipes`, `ssh_keys`, `database_schemas`, `database_users`, `teams`,
`nginx_templates`, `firewall_rules`. The remaining resources share the same
helpers and patterns but have not each been individually exercised against a
live account; `sites` and `servers` in particular carry many nested/computed
fields and may need refinement against real responses.

Resource names follow the generated schema, e.g. `laravelforge_servers`,
`laravelforge_sites`, `laravelforge_ssh_keys`, `laravelforge_database_schemas`.

## Generating the provider

The generation pipeline is driven by the `GNUmakefile`:

```sh
# Install the pinned code generators (tfplugingen-openapi, tfplugingen-framework).
make tools

# (Optional) Download the latest Forge OpenAPI document.
make fetch-spec

# Regenerate the provider code spec and the resource schemas.
make generate
```

`make generate` runs the following steps:

1. **Preprocess** (`generator/preprocess`) rewrites the raw OpenAPI document
   (`generator/docs.openapi.json`) into a generator-friendly form
   (`generator/docs.openapi.processed.json`). The Forge API speaks JSON:API, so
   this step:
   - unwraps the `{ "data": { ... } }` response/relationship envelopes,
   - collapses JSON:API resource-identifier objects (`{id, type}`) down to a
     plain string id, and
   - renames request properties that would collide with reserved Terraform
     attribute names (e.g. `provider` → `cloud_provider`).

   These rewrites also avoid duplicate `DataValue`/`DataType` Go types that the
   framework generator would otherwise emit (it does not namespace nested object
   types by their parent).

2. **`tfplugingen-openapi`** turns the processed spec plus
   `generator/generator_config.yml` into the provider code specification
   (`provider_code_spec.json`).

3. **`tfplugingen-framework`** generates the resource schemas and models
   (`internal/provider/resource_*/*_resource_gen.go`).

4. **Scaffold** (`generator/scaffold`) writes a hand-editable
   `<name>_resource.go` for any resource that does not yet have one. These
   implement `Metadata` and `Schema` (delegating to the generated schema) and
   leave the CRUD methods as TODO stubs. The scaffolder is idempotent and never
   overwrites an existing implementation.

> **Note:** the OpenAPI generators only produce schemas and models. The CRUD
> logic against the Forge API (likely via `forge-go-sdk`) is implemented by hand
> in the scaffolded `*_resource.go` files.
