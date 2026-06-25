# Resource CRUD patterns

Every resource follows the same shape. Copy an existing implementation that
matches the resource's traits rather than starting blank:

- `internal/provider/resource_recipes/recipes_resource.go` — org-level, **sync**
  create (POST returns the body), update, import.
- `internal/provider/resource_ssh_keys/ssh_keys_resource.go` — server-scoped,
  **async** create (202 empty body → resolve id by listing), no update
  (everything `RequiresReplace`), write-only field, import.
- `internal/provider/resource_database_users/database_users_resource.go` —
  async create **and** async update (PUT then re-read), list field, sensitive
  field.
- `internal/provider/resource_teams/teams_resource.go` — nested list inputs
  (`invites`/`users`) serialized with `providerutil.AttrToAny`.
- `internal/provider/resource_php_opcache/php_opcache_resource.go` — singleton
  (no self-id; read path == create path; no `ResolveCreatedID`).
- `internal/provider/resource_deployments/deployments_resource.go` — action
  resource (create-only; Update is a state passthrough, Delete is a no-op).

## The skeleton

```go
package resource_<name>

var (
	_ resource.Resource                = (*<name>Resource)(nil)
	_ resource.ResourceWithConfigure   = (*<name>Resource)(nil)
	_ resource.ResourceWithImportState = (*<name>Resource)(nil)
)

func New<Prefix>Resource() resource.Resource { return &<name>Resource{} }

type <name>Resource struct{ client *forge.Client }

func (r *<name>Resource) Metadata(...) { resp.TypeName = req.ProviderTypeName + "_<name>" }
func (r *<name>Resource) Configure(...) { r.client = providerutil.Client(req, resp) }
func (r *<name>Resource) Schema(ctx, _, resp) {
	s := <Prefix>ResourceSchema(ctx)
	// ... schema tuning (below) ...
	resp.Schema = s
}
// Create / Read / Update / Delete / ImportState
// + a <x>Data struct and (d *<x>Data) apply(m *<Prefix>Model)
```

Look up the exact model field names/types in the package's
`*_resource_gen.go` (`<Prefix>Model`), and each attribute's
Required/Optional/Computed flags.

## Schema tuning (`providerutil` helpers)

The generator can't know intent, so adjust the generated schema in `Schema()`:

- **`Required(s.Attributes, name)`** — parent path params (`organization`,
  `server`, `site`, …) come out Optional+Computed but are needed to address the
  resource and are immutable; this makes them Required + RequiresReplace.
- **`Input(s.Attributes, names...)`** — write-only request fields the API never
  echoes come out Optional+Computed, which leaves them unknown after apply
  ("Provider produced invalid result"). This makes them optional-only so unset =
  null. Covers nested/list attributes too.
- **`RequiresReplace(s.Attributes, names...)`** — fields with no in-place update.
  Rule: if the resource has no update endpoint, mark every create-input field;
  if it does, mark only the create fields absent from the update body.
- **`Sensitive(s.Attributes, names...)`** — passwords, tokens, keys.
- **`UseStateForUnknown(s.Attributes, "id", "<selfid>")`** — the `id` and the
  self-id path-param field (e.g. `monitor`, `job`, `database`).
- **`AttrToAny(value)`** — converts a framework value (incl. nested objects and
  lists) to a plain Go value for a request body. Use it for nested-block inputs
  (cloud-provider blocks on servers, `invites`/`users` on teams).

## Create

The pivot is **sync vs async**, determined by the create response: a `200/201`
with a body is sync; a `202` with an empty body is async. Check the spec.

**Sync:**
```go
var data <x>Data
err := r.client.Do(ctx, http.MethodPost, collection, body, &data)
data.apply(&plan)
```

**Async** (the API assigns the id but returns nothing, so resolve it by diffing
the collection listing):
```go
before, _ := providerutil.ListIDs(ctx, r.client, collection)
r.client.Do(ctx, http.MethodPost, collection, body, nil)
id, _ := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data)
data.apply(&plan)
```
Pass `""` as the name when the resource has no `name` field — `ResolveCreatedID`
falls back to "the single new id".

Build `body` as `map[string]any`, skipping null/unknown values. The path-param
parents come from the model (mind types: `organization` is a string, `server`/
`site` are int64 — `%s` vs `%d`).

## Read / Update / Delete

- **Read**: GET the item, `data.apply(&state)`. On `forge.NotFound(err)`, call
  `resp.State.RemoveResource(ctx)` so Terraform sees it as gone.
- **Update**: build the update body, PUT, then **GET and re-apply** (the update
  may be async, so don't trust the PUT response body). If there's no update
  endpoint, implement Update as `Get(plan)` → `Set(plan)`; it won't be invoked
  because everything is `RequiresReplace`.
- **Delete**: DELETE the item; ignore `forge.NotFound`. Action resources with no
  delete endpoint just return (removed from state only).

## `apply` — mapping the JSON:API wire format

The response is nested. Define `<x>Data` to match the wire shape and copy onto
the flat model. Never overwrite write-only inputs (passwords, keys, content the
API doesn't return) — leave the plan/state value:

```go
type <x>Data struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string  `json:"name"`
		Status    string  `json:"status"`
		CreatedBy *int64  `json:"created_by"` // pointer for nullable
	} `json:"attributes"`
}

func (d *<x>Data) apply(m *<Prefix>Model) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	// self-id int64 from the string id:
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.<SelfId> = types.Int64Value(id)
	}
}
```

Every Computed-only attribute must get a known value here (from the response, or
`types.XNull()` if absent) — otherwise apply fails with "invalid result object".

## ImportState

```go
parts, err := providerutil.ParseID(req.ID, <numParents+1>)
// SetAttribute each parent (ParseInt64 where the model field is Int64) and "id".
```
The import ID is `parent1/parent2/.../id`, e.g. `my-org/123456/789`.

## Schema validity smoke test

Drop this in `internal/provider/` temporarily to catch schema errors across all
resources, run it, then delete it:

```go
func TestZZSchemaValid(t *testing.T) {
	srv := providerserver.NewProtocol6(New("test")())()
	resp, _ := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("%s - %s", d.Summary, d.Detail)
		}
	}
	t.Logf("%d resource schemas", len(resp.ResourceSchemas))
}
```

## Common failure → cause

- `DataValue redeclared` / stray `data` attribute → envelope not unwrapped
  (preprocessing).
- `"provider" is a reserved field name` → add to the `renames` map.
- `invalid result object after apply: <attr> still unknown` → a write-only input
  needs `Input()`, or a Computed field isn't set in `apply`.
- `inconsistent result after apply: <attr> changed` → `apply` overwrote a
  configured input with a differing response value; leave inputs from plan.
- `json: cannot unmarshal array into ...Data` → an async id wasn't resolved, so
  a GET hit the collection (array) instead of an item.
