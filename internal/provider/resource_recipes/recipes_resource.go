package resource_recipes

// This file implements the recipes resource. Unlike *_resource_gen.go it is NOT
// regenerated, so the CRUD logic below is safe to edit.

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
	_ resource.Resource                = (*recipesResource)(nil)
	_ resource.ResourceWithConfigure   = (*recipesResource)(nil)
	_ resource.ResourceWithImportState = (*recipesResource)(nil)
)

// NewRecipesResource returns a new recipes resource.
func NewRecipesResource() resource.Resource {
	return &recipesResource{}
}

type recipesResource struct {
	client *forge.Client
}

func (r *recipesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recipes"
}

func (r *recipesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *recipesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := RecipesResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	// name, user and script are updatable in place; team_id is a write-only
	// input honored only at creation time.
	providerutil.Input(s.Attributes, "team_id")
	providerutil.RequiresReplace(s.Attributes, "team_id")
	providerutil.UseStateForUnknown(s.Attributes, "id", "recipe")

	s.MarkdownDescription = "Manages a recipe in the organization."
	resp.Schema = s
}

func (r *recipesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RecipesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":   plan.Name.ValueString(),
		"user":   plan.User.ValueString(),
		"script": plan.Script.ValueString(),
	}
	if !plan.TeamId.IsNull() && !plan.TeamId.IsUnknown() {
		body["team_id"] = plan.TeamId.ValueString()
	}

	var data recipeData
	endpoint := fmt.Sprintf("/orgs/%s/recipes", plan.Organization.ValueString())
	if err := r.client.Do(ctx, http.MethodPost, endpoint, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to create recipe", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recipesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RecipesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data recipeData
	endpoint := fmt.Sprintf("/orgs/%s/recipes/%s", state.Organization.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read recipe", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *recipesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RecipesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":   plan.Name.ValueString(),
		"user":   plan.User.ValueString(),
		"script": plan.Script.ValueString(),
	}

	var data recipeData
	endpoint := fmt.Sprintf("/orgs/%s/recipes/%s", plan.Organization.ValueString(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to update recipe", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recipesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RecipesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/recipes/%s", state.Organization.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete recipe", err.Error())
	}
}

func (r *recipesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/recipe\": %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// recipeData is the JSON:API representation of a recipe.
type recipeData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		User      string `json:"user"`
		Script    string `json:"script"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

func (d *recipeData) apply(m *RecipesModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.User = types.StringValue(d.Attributes.User)
	m.Script = types.StringValue(d.Attributes.Script)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Recipe = types.Int64Value(id)
	}
}
