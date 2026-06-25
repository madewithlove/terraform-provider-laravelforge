package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_background_processes"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_composer_credentials"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_database_schemas"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_database_users"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_deployment_webhooks"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_deployments"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_domain_certificates"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_firewall_rules"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_heartbeats"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_monitors"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_nginx_templates"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_php_opcache"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_php_versions"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_recipe_runs"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_recipes"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_redirect_rules"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_region_vpcs"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_roles"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_security_rules"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_server_scheduled_jobs"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_servers"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_site_commands"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_site_domains"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_site_scheduled_jobs"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_sites"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_ssh_keys"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_team_invites"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/provider/resource_teams"
)

type LaravelforgeProvider struct {
	version string
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LaravelforgeProvider{version: version}
	}
}

func (p *LaravelforgeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "laravelforge"
	resp.Version = p.version
}

// LaravelforgeProviderModel maps the provider configuration block.
type LaravelforgeProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIToken types.String `tfsdk:"api_token"`
}

func (p *LaravelforgeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage [Laravel Forge](https://forge.laravel.com) resources.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Base URL of the Forge API. Defaults to `" + forge.DefaultEndpoint + "`. May also be set via the `FORGE_ENDPOINT` environment variable.",
			},
			"api_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Forge API token. May also be set via the `FORGE_API_TOKEN` environment variable.",
			},
		},
	}
}

func (p *LaravelforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config LaravelforgeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values take precedence over environment variables.
	endpoint := os.Getenv("FORGE_ENDPOINT")
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	token := os.Getenv("FORGE_API_TOKEN")
	if !config.APIToken.IsNull() {
		token = config.APIToken.ValueString()
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Forge API token",
			"Set the api_token provider argument or the FORGE_API_TOKEN environment variable.",
		)
		return
	}

	client := forge.New(endpoint, token)
	resp.ResourceData = client
	resp.DataSourceData = client
}

// Resources returns the resources implemented by the provider. The schemas are
// generated from the Forge OpenAPI document; CRUD is implemented per resource.
func (p *LaravelforgeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resource_background_processes.NewBackgroundProcessesResource,
		resource_composer_credentials.NewComposerCredentialsResource,
		resource_database_schemas.NewDatabaseSchemasResource,
		resource_database_users.NewDatabaseUsersResource,
		resource_deployment_webhooks.NewDeploymentWebhooksResource,
		resource_deployments.NewDeploymentsResource,
		resource_domain_certificates.NewDomainCertificatesResource,
		resource_firewall_rules.NewFirewallRulesResource,
		resource_heartbeats.NewHeartbeatsResource,
		resource_monitors.NewMonitorsResource,
		resource_nginx_templates.NewNginxTemplatesResource,
		resource_php_opcache.NewPhpOpcacheResource,
		resource_php_versions.NewPhpVersionsResource,
		resource_recipe_runs.NewRecipeRunsResource,
		resource_recipes.NewRecipesResource,
		resource_redirect_rules.NewRedirectRulesResource,
		resource_region_vpcs.NewRegionVpcsResource,
		resource_roles.NewRolesResource,
		resource_security_rules.NewSecurityRulesResource,
		resource_server_scheduled_jobs.NewServerScheduledJobsResource,
		resource_servers.NewServersResource,
		resource_site_commands.NewSiteCommandsResource,
		resource_site_domains.NewSiteDomainsResource,
		resource_site_scheduled_jobs.NewSiteScheduledJobsResource,
		resource_sites.NewSitesResource,
		resource_ssh_keys.NewSshKeysResource,
		resource_team_invites.NewTeamInvitesResource,
		resource_teams.NewTeamsResource,
	}
}

func (p *LaravelforgeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
