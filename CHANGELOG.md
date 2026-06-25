# Changelog

All notable changes to this provider are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/), and the provider follows
[Semantic Versioning](https://semver.org/).

## [Unreleased] — 1.0.0

> ⚠️ **BREAKING CHANGE — this release is not backward compatible with `0.1.x`.**
>
> The provider has been rewritten against Laravel Forge's new
> **organization-scoped API** (`https://forge.laravel.com/api`, JSON:API). The
> previous `0.x` releases targeted the legacy Forge v1 API and a different SDK.
> Every resource name, schema, and authentication mechanism has changed. There
> is **no automatic migration path** from `0.1.x` state — plan a re-import.

### Added

- Resources for the full organization API, generated from the Forge OpenAPI
  document and backed by hand-written CRUD: servers, sites, ssh_keys,
  database_schemas, database_users, recipes, teams, team_invites, roles,
  firewall_rules, monitors, nginx_templates, php_versions, php_opcache,
  background_processes, server/site scheduled jobs, site_domains,
  redirect_rules, security_rules, heartbeats, composer_credentials,
  deployment_webhooks, deployments, recipe_runs, region_vpcs, site_commands,
  domain_certificates.
- Provider configuration via `api_token` / `endpoint` (or the `FORGE_API_TOKEN`
  / `FORGE_ENDPOINT` environment variables).
- `terraform import` support for every resource.
- A reproducible code-generation pipeline (`make generate`).

### Changed

- Authentication is now an organization API token (HTTP bearer), not the legacy
  account API.
- All resources are organization-scoped (most require an `organization` and,
  where relevant, a `server`/`site`).

### Removed

- The legacy `0.x` resources and the `madewithlove/forge-go-sdk` dependency.

## How versioning works

This provider is published to the [Terraform Registry](https://registry.terraform.io/),
which derives each version from a Git tag pushed to this repository (the
`Release` workflow runs GoReleaser on tags matching `v*`). Consumers pin it with
a version constraint, e.g.:

```hcl
terraform {
  required_providers {
    laravelforge = {
      source  = "madewithlove/laravelforge"
      version = "~> 1.0"
    }
  }
}
```

Because this rewrite breaks compatibility with `0.1.x`, it must be released as a
new **major** version (`1.0.0`) so that existing `~> 0.1` constraints do not
silently upgrade into it. Cut the release by tagging the merge commit:

```sh
git tag v1.0.0
git push origin v1.0.0
```
