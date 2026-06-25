package resource_sites

// This file implements the sites resource. Unlike *_resource_gen.go it is NOT
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
	_ resource.Resource                = (*sitesResource)(nil)
	_ resource.ResourceWithConfigure   = (*sitesResource)(nil)
	_ resource.ResourceWithImportState = (*sitesResource)(nil)
)

// NewSitesResource returns a new sites resource.
func NewSitesResource() resource.Resource {
	return &sitesResource{}
}

type sitesResource struct {
	client *forge.Client
}

func (r *sitesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sites"
}

func (r *sitesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *sitesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := SitesResourceSchema(ctx)

	// organization and server are path parameters required to build every
	// request; server is emitted Computed-only by the generator so promote it.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")

	// Only root_directory, web_directory, branch, php_version, push_to_deploy
	// and type can be changed in place via the PUT endpoint. Every other
	// create-input field is immutable and forces replacement on change.
	providerutil.RequiresReplace(s.Attributes,
		"allow_wildcard_subdomains",
		"database_id",
		"database_user_id",
		"domain_mode",
		"frontend_build_command",
		"frontend_package_manager",
		"generate_deploy_key",
		"install_composer_dependencies",
		"is_isolated",
		"isolated_user",
		"name",
		"nginx_template_id",
		"nuxt_next_mode",
		"nuxt_next_port",
		"private_deploy_key",
		"public_deploy_key",
		"repository",
		"shared_paths",
		"source_control_provider",
		"statamic_setup",
		"statamic_starter_kit",
		"statamic_super_user_email",
		"statamic_super_user_password",
		"tags",
		"www_redirect_type",
		"zero_downtime_deployments",
	)

	providerutil.Sensitive(s.Attributes, "private_deploy_key", "statamic_super_user_password")
	providerutil.UseStateForUnknown(s.Attributes, "id", "site")

	s.MarkdownDescription = "Add a new site to the server."
	resp.Schema = s
}

func (r *sitesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SitesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"type": plan.Type.ValueString(),
	}
	addStr(body, "allow_wildcard_subdomains", plan.AllowWildcardSubdomains)
	addStr(body, "branch", plan.Branch)
	addInt(body, "database_id", plan.DatabaseId)
	addStr(body, "database_user_id", plan.DatabaseUserId)
	addStr(body, "domain_mode", plan.DomainMode)
	addStr(body, "frontend_build_command", plan.FrontendBuildCommand)
	addStr(body, "frontend_package_manager", plan.FrontendPackageManager)
	addBool(body, "generate_deploy_key", plan.GenerateDeployKey)
	addBool(body, "install_composer_dependencies", plan.InstallComposerDependencies)
	addBool(body, "is_isolated", plan.IsIsolated)
	addStr(body, "isolated_user", plan.IsolatedUser)
	addStr(body, "name", plan.Name)
	addInt(body, "nginx_template_id", plan.NginxTemplateId)
	addStr(body, "nuxt_next_mode", plan.NuxtNextMode)
	addInt(body, "nuxt_next_port", plan.NuxtNextPort)
	addStr(body, "php_version", plan.PhpVersion)
	addStr(body, "private_deploy_key", plan.PrivateDeployKey)
	addStr(body, "public_deploy_key", plan.PublicDeployKey)
	addBool(body, "push_to_deploy", plan.PushToDeploy)
	addStr(body, "repository", plan.Repository)
	addStr(body, "root_directory", plan.RootDirectory)
	if v := providerutil.AttrToAny(plan.SharedPaths); v != nil {
		body["shared_paths"] = v
	}
	addStr(body, "source_control_provider", plan.SourceControlProvider)
	addStr(body, "statamic_setup", plan.StatamicSetup)
	addStr(body, "statamic_starter_kit", plan.StatamicStarterKit)
	addStr(body, "statamic_super_user_email", plan.StatamicSuperUserEmail)
	addStr(body, "statamic_super_user_password", plan.StatamicSuperUserPassword)
	if v := providerutil.AttrToAny(plan.Tags); v != nil {
		body["tags"] = v
	}
	addStr(body, "web_directory", plan.WebDirectory)
	addStr(body, "www_redirect_type", plan.WwwRedirectType)
	addBool(body, "zero_downtime_deployments", plan.ZeroDowntimeDeployments)

	org := plan.Organization.ValueString()
	server := plan.Server.ValueString()
	collection := fmt.Sprintf("/orgs/%s/servers/%s/sites", org, server)

	// Creation is asynchronous (202, no body); snapshot existing sites, create,
	// then resolve the new site by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list sites", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create site", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created site", err.Error())
		return
	}

	var data siteData
	if err := r.client.Do(ctx, http.MethodGet, fmt.Sprintf("/orgs/%s/sites/%s", org, id), nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created site", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sitesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SitesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data siteData
	endpoint := fmt.Sprintf("/orgs/%s/sites/%s", state.Organization.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read site", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sitesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SitesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The update endpoint accepts a different set of keys than create; only the
	// six in-place updatable fields are sent.
	body := map[string]any{}
	addStr(body, "root_path", plan.RootDirectory)
	addStr(body, "directory", plan.WebDirectory)
	addStr(body, "repository_branch", plan.Branch)
	addStr(body, "php_version", plan.PhpVersion)
	addBool(body, "push_to_deploy", plan.PushToDeploy)
	addStr(body, "type", plan.Type)

	org := plan.Organization.ValueString()
	server := plan.Server.ValueString()
	id := plan.Id.ValueString()
	writeItem := fmt.Sprintf("/orgs/%s/servers/%s/sites/%s", org, server, id)
	if err := r.client.Do(ctx, http.MethodPut, writeItem, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update site", err.Error())
		return
	}

	// The update may be applied asynchronously; re-read to refresh state.
	var data siteData
	if err := r.client.Do(ctx, http.MethodGet, fmt.Sprintf("/orgs/%s/sites/%s", org, id), nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated site", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sitesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SitesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	writeItem := fmt.Sprintf("/orgs/%s/servers/%s/sites/%s",
		state.Organization.ValueString(), state.Server.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, writeItem, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete site", err.Error())
	}
}

// ImportState supports `terraform import` using the
// "organization/server/site" identifier form.
func (r *sitesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site\": %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// addStr adds a string value to the request body when it is set.
func addStr(body map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		body[key] = v.ValueString()
	}
}

// addInt adds an int64 value to the request body when it is set.
func addInt(body map[string]any, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		body[key] = v.ValueInt64()
	}
}

// addBool adds a bool value to the request body when it is set.
func addBool(body map[string]any, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		body[key] = v.ValueBool()
	}
}

// siteData is the JSON:API representation of a site. Only the id and the raw
// attributes map are decoded; individual computed outputs are pulled out in
// apply via the get* helpers.
type siteData struct {
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes"`
}

// apply copies the API response onto the Terraform model. It sets the id and
// self-id, then fills every Computed-only output attribute with a known value
// (Null when the response omits it) so the post-apply state is never unknown.
//
// Optional input fields (name, php_version, branch, root_directory, etc.) are
// intentionally left at their plan/state values to avoid "inconsistent result
// after apply" and unnecessary drift on fields the API normalizes.
func (d *siteData) apply(m *SitesModel) {
	m.Id = types.StringValue(d.ID)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Site = types.Int64Value(id)
	}

	attrs := d.Attributes

	// Computed-only scalar outputs.
	m.AppType = getStr(attrs, "app_type")
	m.CreatedAt = getStr(attrs, "created_at")
	m.Database = getStr(attrs, "database")
	m.DeploymentScript = getStr(attrs, "deployment_script")
	m.DeploymentStatus = getStr(attrs, "deployment_status")
	m.DeploymentUrl = getStr(attrs, "deployment_url")
	m.HealthcheckUrl = getStr(attrs, "healthcheck_url")
	m.Https = getBool(attrs, "https")
	m.Isolated = getBool(attrs, "isolated")
	m.LatestDeployment = getStr(attrs, "latest_deployment")
	m.QuickDeploy = getBool(attrs, "quick_deploy")
	m.Status = getStr(attrs, "status")
	m.UpdatedAt = getStr(attrs, "updated_at")
	m.Url = getStr(attrs, "url")
	m.User = getStr(attrs, "user")
	m.UsesEnvoyer = getBool(attrs, "uses_envoyer")
	m.Wildcards = getBool(attrs, "wildcards")

	// server is a required input; keep the configured value but fall back to the
	// response when it is otherwise unknown (e.g. during import).
	if m.Server.IsNull() || m.Server.IsUnknown() {
		m.Server = getStr(attrs, "server")
	}

	// Computed-only list outputs. We do not currently surface their contents, so
	// expose them as known-null string lists to keep the state consistent.
	m.Aliases = types.ListNull(types.StringType)
	m.RedirectRules = types.ListNull(types.StringType)
	m.SecurityRules = types.ListNull(types.StringType)

	// Computed-only single-nested output. Use the generated null constructor so
	// the value is known rather than "(known after apply)".
	m.MaintenanceMode = NewMaintenanceModeValueNull()
}

// getStr returns attrs[key] as a known string value, or null when absent or not
// a string.
func getStr(attrs map[string]any, key string) types.String {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return types.StringValue(s)
		}
	}
	return types.StringNull()
}

// getBool returns attrs[key] as a known bool value, or null when absent or not a
// bool.
func getBool(attrs map[string]any, key string) types.Bool {
	if v, ok := attrs[key]; ok {
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
	}
	return types.BoolNull()
}
