package resource_monitors

// This file implements the monitors resource. Unlike *_resource_gen.go it is NOT
// regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

var (
	_ resource.Resource                = (*monitorsResource)(nil)
	_ resource.ResourceWithConfigure   = (*monitorsResource)(nil)
	_ resource.ResourceWithImportState = (*monitorsResource)(nil)
)

// NewMonitorsResource returns a new monitors resource.
func NewMonitorsResource() resource.Resource {
	return &monitorsResource{}
}

type monitorsResource struct {
	client *forge.Client
}

func (r *monitorsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitors"
}

func (r *monitorsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *monitorsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := MonitorsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	// Monitors have no update endpoint, so every create-body field forces
	// replacement.
	providerutil.RequiresReplace(s.Attributes, "minutes", "notify", "operator", "threshold", "type")
	providerutil.UseStateForUnknown(s.Attributes, "monitor", "id")

	resp.Schema = s
}

func (r *monitorsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MonitorsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"type":     plan.Type.ValueString(),
		"operator": plan.Operator.ValueString(),
		"notify":   plan.Notify.ValueString(),
	}
	if !plan.Threshold.IsNull() && !plan.Threshold.IsUnknown() {
		f, _ := plan.Threshold.ValueBigFloat().Float64()
		body["threshold"] = f
	}
	if !plan.Minutes.IsNull() && !plan.Minutes.IsUnknown() {
		body["minutes"] = plan.Minutes.ValueInt64()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/monitors",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list monitors", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create monitor", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created monitor", err.Error())
		return
	}

	var data monitorData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created monitor", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MonitorsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data monitorData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/monitors/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read monitor", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: every configurable attribute is marked RequiresReplace.
func (r *monitorsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MonitorsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MonitorsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/monitors/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete monitor", err.Error())
	}
}

func (r *monitorsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/monitor\": %s.", err))
		return
	}
	server, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Server must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), server)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// monitorData is the JSON:API representation of a monitor.
type monitorData struct {
	ID         string `json:"id"`
	Attributes struct {
		Type           string   `json:"type"`
		Operator       string   `json:"operator"`
		Threshold      *float64 `json:"threshold"`
		Minutes        *int64   `json:"minutes"`
		Notify         string   `json:"notify"`
		State          string   `json:"state"`
		StateChangedAt *string  `json:"state_changed_at"`
		Status         string   `json:"status"`
		CreatedAt      string   `json:"created_at"`
		UpdatedAt      string   `json:"updated_at"`
	} `json:"attributes"`
}

func (d *monitorData) apply(m *MonitorsModel) {
	m.Id = types.StringValue(d.ID)
	m.Type = types.StringValue(d.Attributes.Type)
	m.Operator = types.StringValue(d.Attributes.Operator)
	m.Notify = types.StringValue(d.Attributes.Notify)
	m.State = types.StringValue(d.Attributes.State)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Threshold != nil {
		m.Threshold = types.NumberValue(big.NewFloat(*d.Attributes.Threshold))
	} else {
		m.Threshold = types.NumberNull()
	}
	if d.Attributes.Minutes != nil {
		m.Minutes = types.Int64Value(*d.Attributes.Minutes)
	} else {
		m.Minutes = types.Int64Null()
	}
	if d.Attributes.StateChangedAt != nil {
		m.StateChangedAt = types.StringValue(*d.Attributes.StateChangedAt)
	} else {
		m.StateChangedAt = types.StringNull()
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Monitor = types.Int64Value(id)
	}
}
