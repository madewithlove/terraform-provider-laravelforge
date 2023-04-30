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
var _ datasource.DataSource = &DeploymentDataSource{}

func NewDeploymentDataSource() datasource.DataSource {
	return &DeploymentDataSource{}
}

// DeploymentDataSource defines the data source implementation.
type DeploymentDataSource struct {
	client *forge.APIClient
}

// DeploymentDataSourceModel describes the data source data model.
type DeploymentDataSourceModel struct {
	Id              types.Int64  `tfsdk:"id"`
	ServerId        types.Int64  `tfsdk:"server_id"`
	SiteId          types.Int64  `tfsdk:"site_id"`
	Type            types.Int64  `tfsdk:"type"`
	CommitHash      types.String `tfsdk:"commit_hash"`
	CommitAuthor    types.String `tfsdk:"commit_author"`
	CommitMessage   types.String `tfsdk:"commit_message"`
	StartedAt       types.String `tfsdk:"started_at"`
	EndedAt         types.String `tfsdk:"ended_at"`
	Status          types.String `tfsdk:"status"`
	DisplayableType types.String `tfsdk:"displayable_type"`
}

func (d *DeploymentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *DeploymentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Deployment data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "",
				Required:            true,
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the server.",
				Required:            true,
			},
			"site_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the site.",
				Required:            true,
			},
			"type": schema.Int64Attribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"commit_hash": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"commit_author": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"commit_message": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"started_at": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"ended_at": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"displayable_type": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
		},
	}
}

func (d *DeploymentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DeploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeploymentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deployment, _, err := d.client.DefaultApi.GetDeployment(ctx, int32(data.ServerId.ValueInt64()), int32(data.SiteId.ValueInt64()), int32(data.Id.ValueInt64()))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment, got error: %s", err))
		return
	}

	data.Type = types.Int64Value(int64(deployment.Type_))
	data.CommitHash = types.StringValue(deployment.CommitHash)
	data.CommitAuthor = types.StringValue(deployment.CommitAuthor)
	data.CommitMessage = types.StringValue(deployment.CommitMessage)
	data.StartedAt = types.StringValue(deployment.StartedAt)
	data.EndedAt = types.StringValue(deployment.EndedAt)
	data.Status = types.StringValue(deployment.Status)
	data.DisplayableType = types.StringValue(deployment.DisplayableType)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
