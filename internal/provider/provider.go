package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"

	ForgeClient "github.com/madewithlove/forge-go-sdk"
)

// Ensure ForgeProvider satisfies various provider interfaces.
var _ provider.Provider = &ForgeProvider{}

// ForgeProvider defines the provider implementation.
type ForgeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ForgeProviderModel describes the provider data model.
type ForgeProviderModel struct {
	Token types.String `tfsdk:"token"`
}

func (p *ForgeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "laravelforge"
	resp.Version = p.version
}

func (p *ForgeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "Laravel Forge API token.",
				Optional:            true,
			},
		},
	}
}

func (p *ForgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ForgeProviderModel

	apiToken := os.Getenv("FORGE_API_TOKEN")

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Token.ValueString() != "" {
		apiToken = data.Token.ValueString()
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token Configuration",
			"While configuring the provider, the API token was not found in "+
				"the FORGE_API_TOKEN environment variable or provider "+
				"configuration block token attribute.",
		)
	}

	config := ForgeClient.NewConfiguration()
	config.AddDefaultHeader("Authorization", "Bearer "+apiToken)

	client := ForgeClient.NewAPIClient(config)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ForgeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewResourceServer,
	}
}

func (p *ForgeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDaemonDataSource,
		NewSiteDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ForgeProvider{
			version: version,
		}
	}
}
