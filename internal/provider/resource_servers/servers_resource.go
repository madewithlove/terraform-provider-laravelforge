package resource_servers

// This file implements the servers resource. Unlike *_resource_gen.go it is NOT
// regenerated, so the CRUD logic below is safe to edit.
//
// The Forge API has no update endpoint for servers, so every create-input field
// is marked RequiresReplace; the Update method is therefore never exercised in
// practice.

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
	_ resource.Resource                = (*serversResource)(nil)
	_ resource.ResourceWithConfigure   = (*serversResource)(nil)
	_ resource.ResourceWithImportState = (*serversResource)(nil)
)

// NewServersResource returns a new servers resource.
func NewServersResource() resource.Resource {
	return &serversResource{}
}

type serversResource struct {
	client *forge.Client
}

func (r *serversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (r *serversResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *serversResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := ServersResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")

	// There is no update endpoint, so all create-input fields are immutable and
	// changing any of them forces a replacement.
	providerutil.RequiresReplace(s.Attributes,
		"name", "cloud_provider", "type", "ubuntu_version",
		"add_key_to_source_control", "credential_id", "database", "database_type",
		"php_version", "recipe_id", "tags", "team_id",
		"akamai", "aws", "custom", "hetzner", "laravel", "ocean2", "vultr",
	)

	providerutil.UseStateForUnknown(s.Attributes, "id", "server")

	s.MarkdownDescription = "Manages a Forge server. Supports both standard cloud providers and custom VPS configurations."
	resp.Schema = s
}

func (r *serversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":           plan.Name.ValueString(),
		"cloud_provider": plan.CloudProvider.ValueString(),
		"type":           plan.Type.ValueString(),
		"ubuntu_version": plan.UbuntuVersion.ValueString(),
	}

	if !plan.AddKeyToSourceControl.IsNull() && !plan.AddKeyToSourceControl.IsUnknown() {
		body["add_key_to_source_control"] = plan.AddKeyToSourceControl.ValueBool()
	}
	if !plan.CredentialId.IsNull() && !plan.CredentialId.IsUnknown() {
		body["credential_id"] = plan.CredentialId.ValueString()
	}
	if !plan.Database.IsNull() && !plan.Database.IsUnknown() {
		body["database"] = plan.Database.ValueString()
	}
	if !plan.DatabaseType.IsNull() && !plan.DatabaseType.IsUnknown() {
		body["database_type"] = plan.DatabaseType.ValueString()
	}
	if !plan.PhpVersion.IsNull() && !plan.PhpVersion.IsUnknown() {
		body["php_version"] = plan.PhpVersion.ValueString()
	}
	if !plan.RecipeId.IsNull() && !plan.RecipeId.IsUnknown() {
		body["recipe_id"] = plan.RecipeId.ValueInt64()
	}
	if !plan.TeamId.IsNull() && !plan.TeamId.IsUnknown() {
		body["team_id"] = plan.TeamId.ValueInt64()
	}
	if v := providerutil.AttrToAny(plan.Tags); v != nil {
		body["tags"] = v
	}

	// Exactly one cloud-provider block is expected; serialize whichever is set.
	if v := providerutil.AttrToAny(plan.Akamai); v != nil {
		body["akamai"] = v
	}
	if v := providerutil.AttrToAny(plan.Aws); v != nil {
		body["aws"] = v
	}
	if v := providerutil.AttrToAny(plan.Custom); v != nil {
		body["custom"] = v
	}
	if v := providerutil.AttrToAny(plan.Hetzner); v != nil {
		body["hetzner"] = v
	}
	if v := providerutil.AttrToAny(plan.Laravel); v != nil {
		body["laravel"] = v
	}
	if v := providerutil.AttrToAny(plan.Ocean2); v != nil {
		body["ocean2"] = v
	}
	if v := providerutil.AttrToAny(plan.Vultr); v != nil {
		body["vultr"] = v
	}

	var data serverData
	endpoint := fmt.Sprintf("/orgs/%s/servers", plan.Organization.ValueString())
	if err := r.client.Do(ctx, http.MethodPost, endpoint, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to create server", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data serverData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d", state.Organization.ValueString(), state.Id.ValueInt64())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read server", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update has no corresponding API endpoint. Because every input field is marked
// RequiresReplace, Terraform replaces the resource instead of calling Update, so
// this implementation simply persists the plan.
func (r *serversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serversResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d", state.Organization.ValueString(), state.Id.ValueInt64())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete server", err.Error())
	}
}

func (r *serversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server\": %s.", err))
		return
	}
	idInt, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Server ID %q is not a valid integer: %s.", parts[1], err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idInt)...)
}

// serverData is the JSON:API representation of a server. The cloud-provider
// blocks and other input fields are preserved from configuration, not mapped
// from the response, so only the purely-computed output fields are captured.
type serverData struct {
	ID         string `json:"id"`
	Attributes struct {
		ConnectionStatus *string `json:"connection_status"`
		CreatedAt        *string `json:"created_at"`
		DbStatus         *string `json:"db_status"`
		Identifier       *string `json:"identifier"`
		IpAddress        *string `json:"ip_address"`
		IsReady          *bool   `json:"is_ready"`
		LocalPublicKey   *string `json:"local_public_key"`
		OpcacheStatus    *string `json:"opcache_status"`
		PhpCliVersion    *string `json:"php_cli_version"`
		PrivateIpAddress *string `json:"private_ip_address"`
		RedisStatus      *string `json:"redis_status"`
		Region           *string `json:"region"`
		Revoked          *bool   `json:"revoked"`
		Size             *string `json:"size"`
		SshPort          *int64  `json:"ssh_port"`
		Timezone         *string `json:"timezone"`
		UpdatedAt        *string `json:"updated_at"`
	} `json:"attributes"`
}

func (d *serverData) apply(m *ServersModel) {
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Id = types.Int64Value(id)
		m.Server = types.Int64Value(id)
	}

	a := d.Attributes
	m.ConnectionStatus = stringOrNull(a.ConnectionStatus)
	m.CreatedAt = stringOrNull(a.CreatedAt)
	m.DbStatus = stringOrNull(a.DbStatus)
	m.Identifier = stringOrNull(a.Identifier)
	m.IpAddress = stringOrNull(a.IpAddress)
	m.IsReady = boolOrNull(a.IsReady)
	m.LocalPublicKey = stringOrNull(a.LocalPublicKey)
	m.OpcacheStatus = stringOrNull(a.OpcacheStatus)
	m.PhpCliVersion = stringOrNull(a.PhpCliVersion)
	m.PrivateIpAddress = stringOrNull(a.PrivateIpAddress)
	m.RedisStatus = stringOrNull(a.RedisStatus)
	m.Region = stringOrNull(a.Region)
	m.Revoked = boolOrNull(a.Revoked)
	m.Size = stringOrNull(a.Size)
	m.SshPort = int64OrNull(a.SshPort)
	m.Timezone = stringOrNull(a.Timezone)
	m.UpdatedAt = stringOrNull(a.UpdatedAt)
}

func stringOrNull(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func boolOrNull(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

func int64OrNull(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
