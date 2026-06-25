---
name: update-forge-provider
description: >-
  Regenerate and extend the Laravel Forge Terraform provider from its OpenAPI
  document. Use this whenever the Forge API changes, the `docs.openapi.json`
  spec is refreshed, a new Forge resource needs a Terraform resource, or
  generated schemas/CRUD need to be rebuilt — even if the request is phrased as
  "the Forge API added X", "regenerate the provider", "the spec changed", or
  "add a <thing> resource". It encodes the spec-preprocessing rules, the CRUD
  implementation pattern, live validation against Forge, and the CI specifics
  that this repo depends on, so the work is repeatable instead of rediscovered.
---

# Updating the Laravel Forge Terraform provider

This provider's resource **schemas** are generated from the Forge OpenAPI
document; the **CRUD** logic is hand-written on top. The generators only emit
schemas + models — never CRUD — so adding or changing a resource is always two
moves: regenerate the schema, then write/adjust the CRUD.

The Forge API is **JSON:API**, which is hostile to the code generators in
specific, recurring ways. The preprocessor (`generator/preprocess`) neutralizes
those before generation. Understanding why it exists is the key to this skill —
see "Preprocessing rules" below and `references/crud-patterns.md` for the rest.

## When the API updates — the loop

1. **Refresh the spec**: `make fetch-spec` (downloads to `generator/docs.openapi.json`).
2. **Diff it** against the committed spec to see what changed — new paths, new
   schemas, renamed/removed fields. `git diff generator/docs.openapi.json`.
3. **Update `generator/generator_config.yml`** if there are new resources to
   expose or path/method changes (one block per resource: create/read/update/delete).
4. **Update the preprocessor** if the new spec introduces any of the recurring
   JSON:API hazards (see "Preprocessing rules").
5. **Regenerate**: `make generate`. This runs preprocess → `tfplugingen-openapi`
   → `tfplugingen-framework` (resources + provider) → scaffold → `gofmt` →
   `go generate` (docs). Tools are pinned; install them once with `make tools`.
6. **Wire any new resources** into `internal/provider/provider.go` `Resources()`
   (add the import + `resource_<name>.New<Prefix>Resource`). The scaffolder
   creates a stub `*_resource.go` but does **not** touch `provider.go`.
7. **Implement/adjust CRUD** for new or changed resources following
   `references/crud-patterns.md`.
8. **Validate** (see "Verifying" and `references/live-validation.md`).
9. **Regenerate docs + examples** for new resources (see "Docs & examples").
10. **Commit** in logical units (pipeline/spec, schemas, CRUD, docs).

The scaffolder is idempotent and never overwrites an existing `*_resource.go`,
so re-running `make generate` is always safe — it only refreshes the generated
`*_resource_gen.go` files and stubs genuinely new resources.

## Preprocessing rules

`generator/preprocess/main.go` rewrites the spec before generation. Each rule
exists because of a concrete failure; when the spec changes, extend the
corresponding table rather than hand-editing generated code:

- **Data-envelope unwrap** — Forge wraps responses and relationships in
  `{ "data": ... }`. Left in, this produces an unidiomatic `data` attribute and
  duplicate `DataValue`/`DataType` Go types that fail to compile. Unwrapped
  automatically by shape; no maintenance needed.
- **Resource-identifier collapse** — JSON:API `{id, type}` objects become plain
  string ids. Automatic by shape.
- **Resource flattening** — `attributes`/`relationships` are hoisted to the top
  level and `type`/`links` dropped, giving flat schemas. Automatic by shape.
- **`renames` map** — Terraform reserves attribute names (`provider`, `count`,
  `for_each`, `depends_on`, `lifecycle`, …). When a request/response field
  collides, add an entry (e.g. `provider` → `cloud_provider`). The map keys are
  component schema names and it also descends into `attributes`.
- **`responseFixes` map** — corrects wrong `$ref`s in the upstream spec (some
  async `202` responses reference the wrong schema). Add `METHOD PATH` → status
  → correct schema entries when you spot a mis-typed generated model.

After regenerating, a field named like a reserved word, a stray `data`
attribute, or a model with fields from an unrelated resource are all signs a
preprocessing table needs a new entry.

## The actual API vs the generated schema

Flattening only shapes the **Terraform schema**. The live API still returns the
nested JSON:API envelope, so CRUD code reads `data.attributes.<field>` and
`data.relationships.<field>` and maps it onto the flat model by hand. This split
is the single most important thing to keep in mind when writing CRUD.

## Verifying

Run all of these; they mirror CI:

```sh
go build ./...
go vet ./...
golangci-lint run ./...          # v2; config is .golangci.yml
go generate ./... && git diff --exit-code   # docs must be committed
```

Also confirm every resource produces a **valid schema** (catches reserved-word
collisions, duplicate types, and unknown-after-apply problems early) by asking
the provider server for its schema and checking for error diagnostics — see the
`GetProviderSchema` smoke-test pattern in `references/crud-patterns.md`.

For real confidence, validate a representative resource against the live API
(`references/live-validation.md`).

## Docs & examples

`go generate` runs `tfplugindocs`, which renders `docs/` from the served schema
plus `examples/`. To make a new resource's page useful:

- Set `s.MarkdownDescription` in its `Schema()` (sourced from the Forge API
  operation description).
- Add `examples/resources/laravelforge_<name>/resource.tf` (required attributes,
  valid enum values from the spec) and `import.sh` (the import ID format the
  resource implements).

Then re-run `go generate` and commit the regenerated `docs/`.

## CI specifics (don't regress these)

- **golangci-lint v2** — the `.golangci.yml` is v2 schema (`version: "2"`,
  `linters.default: none`, `gofmt` under `formatters`, `linters.exclusions.generated`).
  Generated `*_resource_gen.go` files are skipped via their `DO NOT EDIT` header;
  hand-written `*_resource.go` is linted.
- **setup-terraform wrapper** — the `generate` workflow sets
  `terraform_wrapper: false`; otherwise the wrapper corrupts the JSON output
  `tfplugindocs` reads and fails with `invalid character`.
- **Actions on Node 24** — keep the pinned action SHAs current (the repo uses
  Node-24 releases of checkout/setup-go/setup-terraform/golangci-lint-action).

## Versioning

The provider is published to the Terraform Registry from `v*` git tags via
GoReleaser. It follows semver. A spec change that renames/removes
attributes or resources is **breaking** — bump the major version and note it in
`CHANGELOG.md` so existing version constraints don't silently upgrade.
