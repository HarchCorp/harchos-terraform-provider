package provider

import (
	"context"
	"fmt"

	"github.com/HarchCorp/harchos-terraform-provider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
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
	_ datasource.DataSource              = &RegionsDataSource{}
	_ datasource.DataSourceWithConfigure = &RegionsDataSource{}
)

// RegionsDataSource defines the data source implementation.
type RegionsDataSource struct {
	client *client.Client
}

// RegionsDataSourceModel describes the data source data model.
type RegionsDataSourceModel struct {
	ID                   types.String  `tfsdk:"id"`
	Country              types.String  `tfsdk:"country"`
	MinRenewablePercentage types.Float64 `tfsdk:"min_renewable_percentage"`
	MaxCarbonIntensity   types.Float64 `tfsdk:"max_carbon_intensity"`
	Regions              types.List    `tfsdk:"regions"`
}

// RegionModel describes a single region in the data source output.
type RegionModel struct {
	ID                  types.String  `tfsdk:"id"`
	Name                types.String  `tfsdk:"name"`
	Country             types.String  `tfsdk:"country"`
	Continent           types.String  `tfsdk:"continent"`
	CarbonIntensity     types.Float64 `tfsdk:"carbon_intensity"`
	RenewablePercentage types.Float64 `tfsdk:"renewable_percentage"`
	SovereigntyDefault  types.String  `tfsdk:"sovereignty_default"`
	GPUCount            types.Int64   `tfsdk:"gpu_count"`
	ActiveWorkloads     types.Int64   `tfsdk:"active_workloads"`
	PUE                 types.Float64 `tfsdk:"pue"`
	GreenEnergySource   types.String  `tfsdk:"green_energy_source"`
	PricingTier         types.String  `tfsdk:"pricing_tier"`
	GPUHourlyPrice      types.Float64 `tfsdk:"gpu_hourly_price"`
	Status              types.String  `tfsdk:"status"`
}

// NewRegionsDataSource returns a new regions data source.
func NewRegionsDataSource() datasource.DataSource {
	return &RegionsDataSource{}
}

// Metadata returns the data source type name.
func (d *RegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

// Schema returns the data source schema.
func (d *RegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about HarchOS regions. Regions represent " +
			"geographic deployment zones with carbon intensity, renewable energy, GPU pricing, " +
			"and sovereignty metadata. Optionally filter by country, minimum renewable percentage, " +
			"or maximum carbon intensity.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The data source identifier (always \"regions\").",
				Computed:            true,
			},
			"country": schema.StringAttribute{
				MarkdownDescription: "Optional country filter (e.g., \"Morocco\", \"France\", \"UAE\"). " +
					"Returns regions from all countries if not specified.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"min_renewable_percentage": schema.Float64Attribute{
				MarkdownDescription: "Minimum renewable energy percentage filter. " +
					"Only returns regions with renewable_percentage >= this value.",
				Optional: true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
					float64validator.AtMost(100),
				},
			},
			"max_carbon_intensity": schema.Float64Attribute{
				MarkdownDescription: "Maximum carbon intensity filter in gCO2/kWh. " +
					"Only returns regions with carbon_intensity <= this value.",
				Optional: true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "List of HarchOS regions matching the filters.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the region.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the region.",
							Computed:            true,
						},
						"country": schema.StringAttribute{
							MarkdownDescription: "The country where the region is located.",
							Computed:            true,
						},
						"continent": schema.StringAttribute{
							MarkdownDescription: "The continent where the region is located.",
							Computed:            true,
						},
						"carbon_intensity": schema.Float64Attribute{
							MarkdownDescription: "Current carbon intensity in gCO2/kWh.",
							Computed:            true,
						},
						"renewable_percentage": schema.Float64Attribute{
							MarkdownDescription: "Percentage of energy from renewable sources.",
							Computed:            true,
						},
						"sovereignty_default": schema.StringAttribute{
							MarkdownDescription: "Default sovereignty level for the region.",
							Computed:            true,
						},
						"gpu_count": schema.Int64Attribute{
							MarkdownDescription: "Total number of GPUs available in the region.",
							Computed:            true,
						},
						"active_workloads": schema.Int64Attribute{
							MarkdownDescription: "Number of active workloads in the region.",
							Computed:            true,
						},
						"pue": schema.Float64Attribute{
							MarkdownDescription: "Power Usage Effectiveness of the data center.",
							Computed:            true,
						},
						"green_energy_source": schema.StringAttribute{
							MarkdownDescription: "Primary green energy source (e.g., solar, wind, hydro).",
							Computed:            true,
						},
						"pricing_tier": schema.StringAttribute{
							MarkdownDescription: "Pricing tier for the region.",
							Computed:            true,
						},
						"gpu_hourly_price": schema.Float64Attribute{
							MarkdownDescription: "Hourly price per GPU in USD.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The current status of the region.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider-configured client to the data source.
func (d *RegionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read reads the regions data from the API.
func (d *RegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config RegionsDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract filter values
	country := ""
	if !config.Country.IsNull() && !config.Country.IsUnknown() {
		country = config.Country.ValueString()
	}

	minRenewablePct := 0.0
	if !config.MinRenewablePercentage.IsNull() && !config.MinRenewablePercentage.IsUnknown() {
		minRenewablePct = config.MinRenewablePercentage.ValueFloat64()
	}

	maxCarbonIntensity := 0.0
	if !config.MaxCarbonIntensity.IsNull() && !config.MaxCarbonIntensity.IsUnknown() {
		maxCarbonIntensity = config.MaxCarbonIntensity.ValueFloat64()
	}

	// List regions from API
	regions, err := d.client.ListRegions(ctx, country, minRenewablePct, maxCarbonIntensity)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading regions",
			fmt.Sprintf("Could not list regions: %s", err),
		)
		return
	}

	// Map API response to data source model
	regionModels := make([]RegionModel, 0, len(regions))
	for _, region := range regions {
		rm := RegionModel{
			ID:                  types.StringValue(region.ID),
			Name:                types.StringValue(region.Name),
			Country:             types.StringValue(region.Country),
			Continent:           types.StringValue(region.Continent),
			CarbonIntensity:     types.Float64Value(region.CarbonIntensity),
			RenewablePercentage: types.Float64Value(region.RenewablePercentage),
			SovereigntyDefault:  types.StringValue(region.SovereigntyDefault),
			GPUCount:            types.Int64Value(int64(region.GPUCount)),
			ActiveWorkloads:     types.Int64Value(int64(region.ActiveWorkloads)),
			PUE:                 types.Float64Value(region.PUE),
			GreenEnergySource:   types.StringValue(region.GreenEnergySource),
			PricingTier:         types.StringValue(region.PricingTier),
			GPUHourlyPrice:      types.Float64Value(region.GPUHourlyPrice),
			Status:              types.StringValue(region.Status),
		}
		regionModels = append(regionModels, rm)
	}

	config.ID = types.StringValue("regions")

	regionList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                  types.StringType,
			"name":                types.StringType,
			"country":             types.StringType,
			"continent":           types.StringType,
			"carbon_intensity":    types.Float64Type,
			"renewable_percentage": types.Float64Type,
			"sovereignty_default": types.StringType,
			"gpu_count":           types.Int64Type,
			"active_workloads":    types.Int64Type,
			"pue":                 types.Float64Type,
			"green_energy_source": types.StringType,
			"pricing_tier":        types.StringType,
			"gpu_hourly_price":    types.Float64Type,
			"status":              types.StringType,
		},
	}, regionModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Regions = regionList

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "read regions data source", map[string]interface{}{
		"count": len(regions),
	})
}
