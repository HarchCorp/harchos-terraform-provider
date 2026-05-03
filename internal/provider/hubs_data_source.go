package provider

import (
	"context"
	"fmt"

	"github.com/HarchCorp/harchos-terraform-provider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &HubsDataSource{}
	_ datasource.DataSourceWithConfigure = &HubsDataSource{}
)

// HubsDataSource defines the data source implementation.
type HubsDataSource struct {
	client *client.Client
}

// HubsDataSourceModel describes the data source data model.
type HubsDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Region types.String `tfsdk:"region"`
	Hubs   types.List   `tfsdk:"hubs"`
}

// HubModel describes a single hub in the data source output.
type HubModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Region      types.String `tfsdk:"region"`
	Sovereignty types.String `tfsdk:"sovereignty"`
	Capacity    types.Int64  `tfsdk:"capacity"`
	Status      types.String `tfsdk:"status"`
	Tags        types.Map    `tfsdk:"tags"`
}

// NewHubsDataSource returns a new hubs data source.
func NewHubsDataSource() datasource.DataSource {
	return &HubsDataSource{}
}

// Metadata returns the data source type name.
func (d *HubsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hubs"
}

// Schema returns the data source schema.
func (d *HubsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about HarchOS hubs (compute clusters). " +
			"Hubs represent sovereign compute infrastructure where workloads are deployed. " +
			"Optionally filter by region.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The data source identifier (always \"hubs\").",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Optional region filter. Returns hubs from all regions if not specified.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"hubs": schema.ListNestedAttribute{
				MarkdownDescription: "List of HarchOS hubs.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the hub.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the hub.",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "The region where the hub is located.",
							Computed:            true,
						},
						"sovereignty": schema.StringAttribute{
							MarkdownDescription: "The sovereignty level of the hub.",
							Computed:            true,
						},
						"capacity": schema.Int64Attribute{
							MarkdownDescription: "The compute capacity of the hub.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The current status of the hub.",
							Computed:            true,
						},
						"tags": schema.MapAttribute{
							MarkdownDescription: "Tags associated with the hub.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider-configured client to the data source.
func (d *HubsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = providerData.Client
}

// Read reads the hubs data from the API.
func (d *HubsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config HubsDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := ""
	if !config.Region.IsNull() && !config.Region.IsUnknown() {
		region = config.Region.ValueString()
	}

	hubs, err := d.client.ListHubs(ctx, region)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading hubs",
			fmt.Sprintf("Could not list hubs: %s", err),
		)
		return
	}

	// Map API response to data source model
	hubModels := make([]HubModel, 0, len(hubs))
	for _, hub := range hubs {
		hubModel := HubModel{
			ID:          types.StringValue(hub.ID),
			Name:        types.StringValue(hub.Name),
			Region:      types.StringValue(hub.Region),
			Sovereignty: types.StringValue(hub.Sovereignty),
			Capacity:    types.Int64Value(int64(hub.Capacity)),
			Status:      types.StringValue(hub.Status),
			Tags:        frameworkFromStringMap(ctx, hub.Tags),
		}
		hubModels = append(hubModels, hubModel)
	}

	config.ID = types.StringValue("hubs")

	hubList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":          types.StringType,
			"name":        types.StringType,
			"region":      types.StringType,
			"sovereignty": types.StringType,
			"capacity":    types.Int64Type,
			"status":      types.StringType,
			"tags":        types.MapType{ElemType: types.StringType},
		},
	}, hubModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Hubs = hubList

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "read hubs data source", map[string]interface{}{
		"count": len(hubs),
	})
}
