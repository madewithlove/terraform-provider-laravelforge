package resource_firewall_rules

// This file implements the firewall_rules resource. Unlike *_resource_gen.go it
// is NOT regenerated, so the CRUD logic below is safe to edit.

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
	_ resource.Resource                = (*firewallRulesResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallRulesResource)(nil)
	_ resource.ResourceWithImportState = (*firewallRulesResource)(nil)
)

// NewFirewallRulesResource returns a new firewall_rules resource.
func NewFirewallRulesResource() resource.Resource {
	return &firewallRulesResource{}
}

type firewallRulesResource struct {
	client *forge.Client
}

func (r *firewallRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rules"
}

func (r *firewallRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *firewallRulesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := FirewallRulesResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	// Firewall rules have no update endpoint, so every configurable create-body
	// field forces replacement. (ip_address is an empty nested object in the
	// generated schema and is treated as read-only.)
	providerutil.RequiresReplace(s.Attributes, "name", "port", "type")
	providerutil.UseStateForUnknown(s.Attributes, "rule", "id")

	s.MarkdownDescription = "Add a new firewall rule to the server."
	resp.Schema = s
}

func (r *firewallRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"type": plan.Type.ValueString(),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		body["port"] = plan.Port.ValueString()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/firewall-rules",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list firewall rules", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create firewall rule", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created firewall rule", err.Error())
		return
	}

	var data firewallRuleData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created firewall rule", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data firewallRuleData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/firewall-rules/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read firewall rule", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: every configurable attribute is marked RequiresReplace.
func (r *firewallRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/firewall-rules/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete firewall rule", err.Error())
	}
}

func (r *firewallRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/rule\": %s.", err))
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

// firewallRuleData is the JSON:API representation of a firewall rule.
type firewallRuleData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      *string `json:"name"`
		Port      *string `json:"port"`
		Type      string  `json:"type"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	} `json:"attributes"`
}

func (d *firewallRuleData) apply(m *FirewallRulesModel) {
	m.Id = types.StringValue(d.ID)
	m.Type = types.StringValue(d.Attributes.Type)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Name != nil {
		m.Name = types.StringValue(*d.Attributes.Name)
	} else {
		m.Name = types.StringNull()
	}
	if d.Attributes.Port != nil {
		m.Port = types.StringValue(*d.Attributes.Port)
	} else {
		m.Port = types.StringNull()
	}
	// ip_address is an empty nested object in the generated schema; keep it null
	// so the computed attribute resolves to a known value.
	m.IpAddress = NewIpAddressValueNull()
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Rule = types.Int64Value(id)
	}
}
