package resource_region_vpcs

// This file implements the region_vpcs resource. Unlike *_resource_gen.go it is
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
	_ resource.Resource                = (*regionVpcsResource)(nil)
	_ resource.ResourceWithConfigure   = (*regionVpcsResource)(nil)
	_ resource.ResourceWithImportState = (*regionVpcsResource)(nil)
)

// NewRegionVpcsResource returns a new region_vpcs resource.
func NewRegionVpcsResource() resource.Resource {
	return &regionVpcsResource{}
}

type regionVpcsResource struct {
	client *forge.Client
}

func (r *regionVpcsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_region_vpcs"
}

func (r *regionVpcsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *regionVpcsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := RegionVpcsResourceSchema(ctx)

	// organization, credential and region are parent path parameters; region is
	// generated as computed-only but is required to construct the API path.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "credential")
	providerutil.Required(s.Attributes, "region")
	// The API has no update endpoint, so the only configurable attribute (name)
	// forces replacement on change.
	providerutil.RequiresReplace(s.Attributes, "name")
	providerutil.UseStateForUnknown(s.Attributes, "id", "vpc_id")

	s.MarkdownDescription = "Create a private network for the provider."
	resp.Schema = s
}

func (r *regionVpcsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RegionVpcsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}

	collection := fmt.Sprintf("/orgs/%s/server-credentials/%d/regions/%s/vpcs",
		plan.Organization.ValueString(), plan.Credential.ValueInt64(), plan.Region.ValueString())

	// Creation is synchronous and returns the created resource.
	var data regionVpcData
	if err := r.client.Do(ctx, http.MethodPost, collection, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to create region VPC", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *regionVpcsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RegionVpcsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data regionVpcData
	endpoint := fmt.Sprintf("/orgs/%s/server-credentials/%d/regions/%s/vpcs/%s",
		state.Organization.ValueString(), state.Credential.ValueInt64(), state.Region.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read region VPC", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: the only configurable attribute is marked RequiresReplace, so
// Terraform destroys and recreates the VPC instead.
func (r *regionVpcsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RegionVpcsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op: the Laravel Forge API exposes no delete endpoint for
// region VPCs, so the resource is simply removed from Terraform state.
func (r *regionVpcsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *regionVpcsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/credential/region/vpc_id\": %s.", err))
		return
	}
	credential, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Credential must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credential"), credential)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// regionVpcData is the JSON:API representation of a region VPC.
type regionVpcData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string  `json:"name"`
		CidrBlock *string `json:"cidr_block"`
		Subnets   *string `json:"subnets"`
	} `json:"attributes"`
}

// apply copies the API response onto the Terraform model. The organization,
// credential and region path parameters are user-supplied and left untouched.
func (d *regionVpcData) apply(m *RegionVpcsModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	if d.Attributes.CidrBlock != nil {
		m.CidrBlock = types.StringValue(*d.Attributes.CidrBlock)
	} else {
		m.CidrBlock = types.StringNull()
	}
	if d.Attributes.Subnets != nil {
		m.Subnets = types.StringValue(*d.Attributes.Subnets)
	} else {
		m.Subnets = types.StringNull()
	}
	m.VpcId = types.StringValue(d.ID)
}
