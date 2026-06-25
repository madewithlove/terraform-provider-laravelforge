package resource_heartbeats

// This file implements the heartbeats resource. Unlike *_resource_gen.go it is
// NOT regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

var (
	_ resource.Resource                = (*heartbeatsResource)(nil)
	_ resource.ResourceWithConfigure   = (*heartbeatsResource)(nil)
	_ resource.ResourceWithImportState = (*heartbeatsResource)(nil)
)

// NewHeartbeatsResource returns a new heartbeats resource.
func NewHeartbeatsResource() resource.Resource {
	return &heartbeatsResource{}
}

type heartbeatsResource struct {
	client *forge.Client
}

func (r *heartbeatsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeats"
}

func (r *heartbeatsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *heartbeatsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := HeartbeatsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// name, frequency, grace_period and custom_frequency are all updatable in
	// place via the API's PUT endpoint, so none of them force replacement.
	providerutil.UseStateForUnknown(s.Attributes, "id", "heartbeat")

	s.MarkdownDescription = "Manages a heartbeat monitor for a site."
	resp.Schema = s
}

func (r *heartbeatsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HeartbeatsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":         plan.Name.ValueString(),
		"frequency":    plan.Frequency.ValueInt64(),
		"grace_period": plan.GracePeriod.ValueInt64(),
	}
	if !plan.CustomFrequency.IsNull() && !plan.CustomFrequency.IsUnknown() {
		body["custom_frequency"] = plan.CustomFrequency.ValueString()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/heartbeats",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is synchronous and returns the created resource.
	var data heartbeatData
	if err := r.client.Do(ctx, http.MethodPost, collection, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to create heartbeat", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *heartbeatsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HeartbeatsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data heartbeatData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/heartbeats/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read heartbeat", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *heartbeatsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HeartbeatsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":         plan.Name.ValueString(),
		"frequency":    plan.Frequency.ValueInt64(),
		"grace_period": plan.GracePeriod.ValueInt64(),
	}
	if !plan.CustomFrequency.IsNull() && !plan.CustomFrequency.IsUnknown() {
		body["custom_frequency"] = plan.CustomFrequency.ValueString()
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/heartbeats/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update heartbeat", err.Error())
		return
	}

	var data heartbeatData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated heartbeat", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *heartbeatsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HeartbeatsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/heartbeats/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete heartbeat", err.Error())
	}
}

func (r *heartbeatsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/heartbeat\": %s.", err))
		return
	}
	server, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Server must be an integer: %s.", err))
		return
	}
	site, err := providerutil.ParseInt64(parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Site must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), server)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// heartbeatData is the JSON:API representation of a heartbeat.
type heartbeatData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name            string  `json:"name"`
		Frequency       *int64  `json:"frequency"`
		GracePeriod     *int64  `json:"grace_period"`
		CustomFrequency *string `json:"custom_frequency"`
		PingUrl         *string `json:"ping_url"`
		Status          *string `json:"status"`
	} `json:"attributes"`
}

func (d *heartbeatData) apply(m *HeartbeatsModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	if d.Attributes.Frequency != nil {
		m.Frequency = types.Int64Value(*d.Attributes.Frequency)
	}
	if d.Attributes.GracePeriod != nil {
		m.GracePeriod = types.Int64Value(*d.Attributes.GracePeriod)
	}
	if d.Attributes.CustomFrequency != nil {
		m.CustomFrequency = types.StringValue(*d.Attributes.CustomFrequency)
	} else {
		m.CustomFrequency = types.StringNull()
	}
	if d.Attributes.PingUrl != nil {
		m.PingUrl = types.StringValue(*d.Attributes.PingUrl)
	} else {
		m.PingUrl = types.StringNull()
	}
	if d.Attributes.Status != nil {
		m.Status = types.StringValue(*d.Attributes.Status)
	} else {
		m.Status = types.StringNull()
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Heartbeat = types.Int64Value(id)
	}
}
