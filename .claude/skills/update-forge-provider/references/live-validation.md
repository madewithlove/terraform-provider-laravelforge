# Live validation against the Forge API

Compile/vet/lint and the schema smoke test catch structural problems, but only a
real apply confirms the request bodies, async-id resolution, and response
mapping are correct. Validate at least one representative resource end-to-end
after non-trivial changes.

## Prerequisites

- A Forge **API token** with the scopes for the resources under test. Note that
  scopes are fixed at token issuance — editing a token's scopes later does not
  apply to the already-issued token; re-issue it. (Symptom: `403 "Invalid
  scope(s) provided"` on delete even though create worked.)
- The token must belong to an org with an **active subscription**, or creates
  fail with `403 "The organization does not have an active subscription."`
- Keep the token out of git. Put it in a gitignored file (e.g. `.forge_token`)
  and read it into `FORGE_API_TOKEN`.

## Discover org / server without asking

The token can enumerate what it can reach:

```sh
TOKEN=$(tr -d '\n' < .forge_token)
curl -s -H "Authorization: Bearer $TOKEN" -H "Accept: application/json" \
  https://forge.laravel.com/api/orgs        # organizations (use the slug)
curl -s -H "Authorization: Bearer $TOKEN" -H "Accept: application/json" \
  https://forge.laravel.com/api/orgs/<slug>/servers   # servers in an org
```

## Harness (dev override, no install)

```sh
go build -o "$(go env GOPATH)/bin/terraform-provider-laravelforge" .

cat > /tmp/dev.tfrc <<EOF
provider_installation {
  dev_overrides { "madewithlove/laravelforge" = "$(go env GOPATH)/bin" }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE=/tmp/dev.tfrc
export FORGE_API_TOKEN=$(tr -d '\n' < .forge_token)
```

Write a small config under a scratch dir (gitignored, e.g. `examples/validate/`)
and run the full lifecycle:

```sh
terraform apply -auto-approve      # create
terraform plan                     # expect "No changes" (read/drift is correct)
# edit a mutable attr, then:
terraform apply -auto-approve      # update should be in-place, not replace
terraform destroy -auto-approve    # delete
```

After destroy, confirm via the API that the resource is actually gone (deletes
are often async `202` — a brief "removing" status is normal; re-check).

## What to assert

- **Create**: succeeds and state is populated (id known, not null).
- **Read**: a follow-up `plan` shows no changes (mapping round-trips). A computed
  status flipping `installing → installed` is benign, not drift.
- **Update**: changing a mutable attribute is `update in-place`; an immutable one
  forces replacement.
- **Delete**: removed on the API side.

## Good resources to validate

Prefer cheap, side-effect-light resources. `recipes` and `teams` are org-level
and free. `ssh_keys`, `database_schemas`, `database_users`, `nginx_templates`,
`firewall_rules` need an existing server but are cheap. Avoid creating `servers`
(provisions paid infrastructure) unless explicitly authorized.

## Cleanup

Always delete what you created and remove the scratch dir and the dev-override
file. If an apply fails mid-create, the resource may still exist on the API
(orphan) — delete it via `curl -X DELETE`. Quirk to know: Forge counts
**soft-deleted** recipe names in its uniqueness check, so reusing a name you
just deleted can `422` on a later update — use a fresh name when re-testing.
