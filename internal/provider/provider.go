package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	ForgeClient "github.com/madewithlove/forge-go-sdk"
)

// Ensure LaravelForgeProvider satisfies various provider interfaces.
var _ provider.Provider = &LaravelForgeProvider{}

// LaravelForgeProvider defines the provider implementation.
type LaravelForgeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// LaravelForgeProviderModel describes the provider data model.
type LaravelForgeProviderModel struct {
	Token types.String `tfsdk:"token"`
}

func (p *LaravelForgeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "forge"
	resp.Version = p.version
}

func (p *LaravelForgeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "Laravel Forge API token.",
				Optional:            false,
			},
		},
	}
}

func (p *LaravelForgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data LaravelForgeProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := ForgeClient.NewAPIClient(ForgeClient.NewConfiguration())
	//data.Token
	tflog.Trace(ctx, data.Token.String())

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *LaravelForgeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewResourceServer,
	}
}

func (p *LaravelForgeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LaravelForgeProvider{
			version: version,
		}
	}
}
