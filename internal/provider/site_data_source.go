package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/madewithlove/forge-go-sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SiteDataSource{}

func NewSiteDataSource() datasource.DataSource {
	return &SiteDataSource{}
}

// SiteDataSource defines the data source implementation.
type SiteDataSource struct {
	client *forge.APIClient
}

// SiteDataSourceModel describes the data source data model.
type SiteDataSourceModel struct {
	Id                 types.Int64    `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	Aliases            types.ListType `tfsdk:"aliases"`
	Username           types.String   `tfsdk:"username"`
	Directory          types.String   `tfsdk:"directory"`
	Wildcards          types.String   `tfsdk:"wildcards"`
	Status             types.String   `tfsdk:"status"`
	Repository         types.String   `tfsdk:"repository"`
	RepositoryProvider types.String   `tfsdk:"repository_provider"`
	RepositoryBranch   types.String   `tfsdk:"repository_branch"`
	RepositoryStatus   types.String   `tfsdk:"repository_status"`
	QuickDeploy        types.String   `tfsdk:"quick_deploy"`
	ProjectType        types.String   `tfsdk:"project_type"`
	App                types.String   `tfsdk:"app"`
	AppStatus          types.String   `tfsdk:"app_status"`
	SlackChannel       types.String   `tfsdk:"slack_channel"`
	TelegramChatId     types.String   `tfsdk:"telegram_chat_id"`
	TelegramChatTitle  types.String   `tfsdk:"telegram_chat_title"`
	DeploymentUrl      types.String   `tfsdk:"deployment_url"`
	CreatedAt          types.String   `tfsdk:"created_at"`
	Tags               types.ListType `tfsdk:"tags"`
	ServerId           types.Int64    `tfsdk:"server_id"`
}

func (d *SiteDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (d *SiteDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Site data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"aliases": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"directory": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"wildcards": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"repository_provider": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"repository_branch": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"repository_status": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"quick_deploy": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"project_type": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"app_status": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"slack_channel": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"telegram_chat_id": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"telegram_chat_title": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"deployment_url": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"tags": schema.ListAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "",
				Optional:            true,
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "",
				Optional:            true,
			},
		},
	}
}

func (d *SiteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*forge.APIClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *forge.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *SiteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SiteDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	site, _, err := d.client.DefaultApi.GetSite(ctx, int32(data.ServerId.ValueInt64()), int32(data.Id.ValueInt64()))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read site, got error: %s", err))
		return
	}

	data.Name = types.StringValue(string(site.Name))
	//data.Aliases = types.ListValue(types.StringType, site.Aliases)
	data.Username = types.StringValue(string(site.Username))
	data.Directory = types.StringValue(string(site.Directory))
	data.Wildcards = types.StringValue(string(site.Wildcards))
	data.Status = types.StringValue(string(site.Status))
	data.Repository = types.StringValue(string(site.Repository))
	data.RepositoryProvider = types.StringValue(string(site.RepositoryProvider))
	data.RepositoryBranch = types.StringValue(string(site.RepositoryBranch))
	data.RepositoryStatus = types.StringValue(string(site.RepositoryStatus))
	data.QuickDeploy = types.StringValue(string(site.QuickDeploy))
	data.ProjectType = types.StringValue(string(site.ProjectType))
	data.App = types.StringValue(string(site.App))
	data.AppStatus = types.StringValue(string(site.AppStatus))
	data.SlackChannel = types.StringValue(string(site.SlackChannel))
	data.TelegramChatId = types.StringValue(string(site.TelegramChatId))
	data.TelegramChatTitle = types.StringValue(string(site.TelegramChatTitle))
	data.DeploymentUrl = types.StringValue(string(site.DeploymentUrl))
	data.CreatedAt = types.StringValue(string(site.CreatedAt))
	//data.Tags = types.ListValue(types.Int64Type, site.Tags)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
