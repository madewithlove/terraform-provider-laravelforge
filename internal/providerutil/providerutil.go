// Package providerutil holds helpers shared by the per-resource packages:
// retrieving the configured API client and adjusting generated schemas.
package providerutil

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
)

// Client extracts the configured *forge.Client from a Configure request,
// reporting an error diagnostic if the provider data is of an unexpected type.
// It returns nil during the initial (nil ProviderData) configuration pass.
func Client(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *forge.Client {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*forge.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *forge.Client, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return nil
	}
	return client
}

// Required marks an attribute as required and forces replacement when it
// changes. It is used for parent path-parameter and immutable attributes that
// the generator emits as optional/computed.
func Required(attrs map[string]schema.Attribute, name string) {
	switch a := attrs[name].(type) {
	case schema.StringAttribute:
		a.Required, a.Optional, a.Computed = true, false, false
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
		attrs[name] = a
	case schema.Int64Attribute:
		a.Required, a.Optional, a.Computed = true, false, false
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.RequiresReplace())
		attrs[name] = a
	}
}

// Input marks attributes as optional-but-not-computed. The generator emits
// write-only request fields (those never echoed back by the API) as
// Optional+Computed, which leaves them "(known after apply)" and unknown in the
// final state. Making them optional-only means an unset field is simply null.
func Input(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		switch a := attrs[name].(type) {
		case schema.StringAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.Int64Attribute:
			a.Computed = false
			attrs[name] = a
		case schema.BoolAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.Float64Attribute:
			a.Computed = false
			attrs[name] = a
		case schema.ListAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.SetAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.MapAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.ListNestedAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.SetNestedAttribute:
			a.Computed = false
			attrs[name] = a
		case schema.SingleNestedAttribute:
			a.Computed = false
			attrs[name] = a
		}
	}
}

// RequiresReplace forces replacement when the named attribute changes, leaving
// its required/optional/computed flags as generated.
func RequiresReplace(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		switch a := attrs[name].(type) {
		case schema.StringAttribute:
			a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
			attrs[name] = a
		case schema.Int64Attribute:
			a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.RequiresReplace())
			attrs[name] = a
		case schema.BoolAttribute:
			a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.RequiresReplace())
			attrs[name] = a
		case schema.ListAttribute:
			a.PlanModifiers = append(a.PlanModifiers, listplanmodifier.RequiresReplace())
			attrs[name] = a
		case schema.ListNestedAttribute:
			a.PlanModifiers = append(a.PlanModifiers, listplanmodifier.RequiresReplace())
			attrs[name] = a
		case schema.SetNestedAttribute:
			a.PlanModifiers = append(a.PlanModifiers, setplanmodifier.RequiresReplace())
			attrs[name] = a
		case schema.SingleNestedAttribute:
			a.PlanModifiers = append(a.PlanModifiers, objectplanmodifier.RequiresReplace())
			attrs[name] = a
		}
	}
}

// UseStateForUnknown keeps a computed attribute's prior value during planning
// instead of showing "(known after apply)".
func UseStateForUnknown(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		switch a := attrs[name].(type) {
		case schema.StringAttribute:
			a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
			attrs[name] = a
		case schema.Int64Attribute:
			a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
			attrs[name] = a
		}
	}
}

// Sensitive marks the named string attributes as sensitive.
func Sensitive(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		if a, ok := attrs[name].(schema.StringAttribute); ok {
			a.Sensitive = true
			attrs[name] = a
		}
	}
}

// namedItem is the minimal JSON:API shape needed to identify a resource in a
// collection listing.
type namedItem struct {
	ID         string `json:"id"`
	Attributes struct {
		Name string `json:"name"`
	} `json:"attributes"`
}

// ListIDs returns the set of resource ids currently present at listPath.
func ListIDs(ctx context.Context, c *forge.Client, listPath string) (map[string]bool, error) {
	var items []namedItem
	if err := c.Do(ctx, http.MethodGet, listPath, nil, &items); err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(items))
	for _, it := range items {
		ids[it.ID] = true
	}
	return ids, nil
}

// ResolveCreatedID locates the id of a resource created by an asynchronous
// request that returned no body. It lists listPath and returns the id of the
// newly-appeared item, preferring one whose name matches the given name (pass
// "" when the resource has no name attribute). Because creation is
// asynchronous, it retries briefly until a new item appears.
func ResolveCreatedID(ctx context.Context, c *forge.Client, listPath, name string, before map[string]bool) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}

		var items []namedItem
		if err := c.Do(ctx, http.MethodGet, listPath, nil, &items); err != nil {
			lastErr = err
			continue
		}

		var newIDs []string
		var matchNewName, matchName string
		for _, it := range items {
			isNew := !before[it.ID]
			if isNew {
				newIDs = append(newIDs, it.ID)
			}
			if name != "" && it.Attributes.Name == name {
				matchName = it.ID
				if isNew {
					matchNewName = it.ID
				}
			}
		}
		switch {
		case matchNewName != "":
			return matchNewName, nil
		case name == "" && len(newIDs) == 1:
			// No name to match on, but exactly one new resource appeared.
			return newIDs[0], nil
		case attempt == 5 && matchName != "":
			// Last resort: a same-named resource whose id was already known
			// (e.g. recreation of an identical resource).
			return matchName, nil
		case attempt == 5 && len(newIDs) == 1:
			return newIDs[0], nil
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("locating created resource: %w", lastErr)
	}
	return "", fmt.Errorf("created resource did not appear after asynchronous create")
}

// AttrToAny converts a Terraform framework attr.Value into a plain Go value
// suitable for JSON encoding in a request body. Null and unknown values return
// nil (callers should omit them). Nested objects/lists/maps are converted
// recursively. This is used to serialize generated nested-object attributes
// (e.g. cloud-provider blocks) without bespoke per-field code.
func AttrToAny(v attr.Value) any {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil
	}
	switch t := v.(type) {
	case basetypes.StringValue:
		return t.ValueString()
	case basetypes.BoolValue:
		return t.ValueBool()
	case basetypes.Int64Value:
		return t.ValueInt64()
	case basetypes.Float64Value:
		return t.ValueFloat64()
	case basetypes.NumberValue:
		f, _ := t.ValueBigFloat().Float64()
		return f
	case basetypes.ListValue:
		return elementsToAny(t.Elements())
	case basetypes.SetValue:
		return elementsToAny(t.Elements())
	case basetypes.MapValue:
		return attributesToAny(t.Elements())
	case basetypes.ObjectValue:
		return attributesToAny(t.Attributes())
	default:
		// Custom object types (generated *Value) embed basetypes.ObjectValue
		// and expose their members via ToObjectValue.
		if ov, ok := v.(interface {
			ToObjectValue(context.Context) (basetypes.ObjectValue, diag.Diagnostics)
		}); ok {
			obj, _ := ov.ToObjectValue(context.Background())
			return attributesToAny(obj.Attributes())
		}
		return nil
	}
}

func elementsToAny(elems []attr.Value) []any {
	out := make([]any, 0, len(elems))
	for _, e := range elems {
		out = append(out, AttrToAny(e))
	}
	return out
}

func attributesToAny(attrs map[string]attr.Value) map[string]any {
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		if av := AttrToAny(v); av != nil {
			out[k] = av
		}
	}
	return out
}

// ParseID splits a Terraform import ID into the given number of "/"-separated
// parts, returning an error if the count does not match.
func ParseID(id string, parts int) ([]string, error) {
	segments := strings.Split(id, "/")
	if len(segments) != parts {
		return nil, fmt.Errorf("expected %d \"/\"-separated parts, got %d", parts, len(segments))
	}
	return segments, nil
}

// ParseInt64 parses a string segment of an import ID into an int64.
func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
