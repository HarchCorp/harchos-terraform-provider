package provider

import (
	"context"
	"fmt"

	"github.com/HarchCorp/harchos-terraform-provider/internal/client"
	"github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &CarbonAwareScheduleResource{}
	_ resource.ResourceWithConfigure   = &CarbonAwareScheduleResource{}
	_ resource.ResourceWithImportState = &CarbonAwareScheduleResource{}
	_ resource.ResourceWithModifyPlan  = &CarbonAwareScheduleResource{}
)

// CarbonAwareScheduleResource defines the resource implementation.
type CarbonAwareScheduleResource struct {
	client      *client.Client
	region      string
	sovereignty string
}

// CarbonAwareScheduleResourceModel describes the resource data model.
type CarbonAwareScheduleResourceModel struct {
	ID                  types.String  `tfsdk:"id"`
	WorkloadID          types.String  `tfsdk:"workload_id"`
	Enabled             types.Bool    `tfsdk:"enabled"`
	MaxCarbonIntensity  types.Float64 `tfsdk:"max_carbon_intensity"`
	PreferredRegion     types.String  `tfsdk:"preferred_region"`
	DeferWhenHighCarbon types.Bool    `tfsdk:"defer_when_high_carbon"`
	GreenWindowOnly     types.Bool    `tfsdk:"green_window_only"`
	Sovereignty         types.String  `tfsdk:"sovereignty"`
	Status              types.String  `tfsdk:"status"`
	CurrentIntensity    types.Float64 `tfsdk:"current_intensity"`
	RecommendedAction   types.String  `tfsdk:"recommended_action"`
	NextGreenWindow     types.String  `tfsdk:"next_green_window"`
	CreatedAt           types.String  `tfsdk:"created_at"`
	UpdatedAt           types.String  `tfsdk:"updated_at"`
}

// NewCarbonAwareScheduleResource returns a new carbon-aware schedule resource.
func NewCarbonAwareScheduleResource() resource.Resource {
	return &CarbonAwareScheduleResource{}
}

// Metadata returns the resource type name.
func (r *CarbonAwareScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_carbon_aware_schedule"
}

// Schema returns the resource schema.
func (r *CarbonAwareScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a HarchOS carbon-aware schedule. Carbon-aware schedules enable " +
			"automatic workload placement and deferral based on real-time carbon intensity data, " +
			"ensuring compute runs during green energy windows and in regions with lower carbon emissions. " +
			"Sovereignty cannot be downgraded from strict to regional after creation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the carbon-aware schedule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workload_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the workload to apply carbon-aware scheduling to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether carbon-aware scheduling is enabled. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"max_carbon_intensity": schema.Float64Attribute{
				MarkdownDescription: "Maximum carbon intensity threshold in gCO2/kWh. Workloads will be " +
					"deferred if the current intensity exceeds this value. Defaults to 100.",
				Optional: true,
				Computed: true,
				Default:  float64default.StaticFloat64(100),
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
					float64validator.AtMost(1000),
				},
			},
			"preferred_region": schema.StringAttribute{
				MarkdownDescription: "Preferred region for carbon-aware placement. Defaults to \"morocco\".",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("morocco"),
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"defer_when_high_carbon": schema.BoolAttribute{
				MarkdownDescription: "Whether to automatically defer the workload when carbon intensity " +
					"exceeds max_carbon_intensity. Defaults to true.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"green_window_only": schema.BoolAttribute{
				MarkdownDescription: "Whether the workload should only run during identified green windows " +
					"(periods of low carbon intensity). Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"sovereignty": schema.StringAttribute{
				MarkdownDescription: "The sovereignty level for this carbon-aware schedule. Must be one of: " +
					"strict, regional, global. Cannot be downgraded after creation. " +
					"Defaults to the provider sovereignty if not set.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					sovereignty.SovereigntyLevelValidator(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the carbon-aware schedule.",
				Computed:            true,
			},
			"current_intensity": schema.Float64Attribute{
				MarkdownDescription: "The current carbon intensity at the scheduled region in gCO2/kWh.",
				Computed:            true,
			},
			"recommended_action": schema.StringAttribute{
				MarkdownDescription: "The recommended action from the carbon optimization engine " +
					"(schedule_now, defer, no_suitable_hub).",
				Computed: true,
			},
			"next_green_window": schema.StringAttribute{
				MarkdownDescription: "The start time of the next identified green window.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the carbon-aware schedule was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the carbon-aware schedule was last updated.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider-configured client to the resource.
func (r *CarbonAwareScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = providerData.Client
	r.region = providerData.Region
	r.sovereignty = providerData.Sovereignty
}

// ModifyPlan enables sovereignty enforcement at plan time.
func (r *CarbonAwareScheduleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan, state CarbonAwareScheduleResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Sovereignty.IsNull() || plan.Sovereignty.ValueString() == "" {
		resp.Plan.SetAttribute(ctx, path.Root("sovereignty"), r.sovereignty)
	}

	// Sovereignty downgrade prevention
	if !req.State.Raw.IsNull() {
		diags = req.State.Get(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if err := sovereignty.ValidateSovereigntyTransition(state.Sovereignty, plan.Sovereignty); err != nil {
			resp.Diagnostics.AddError(
				"Sovereignty downgrade not allowed",
				fmt.Sprintf("Cannot change carbon-aware schedule sovereignty: %s", err),
			)
		}
	}
}

// Create creates the carbon-aware schedule resource.
func (r *CarbonAwareScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CarbonAwareScheduleResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine effective sovereignty
	sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sovereignty configuration", err.Error())
		return
	}

	// Build the API request
	scheduleReq := &client.CarbonAwareSchedule{
		WorkloadID:          plan.WorkloadID.ValueString(),
		Enabled:             plan.Enabled.ValueBool(),
		MaxCarbonIntensity:  plan.MaxCarbonIntensity.ValueFloat64(),
		PreferredRegion:     plan.PreferredRegion.ValueString(),
		DeferWhenHighCarbon: plan.DeferWhenHighCarbon.ValueBool(),
		GreenWindowOnly:     plan.GreenWindowOnly.ValueBool(),
		Sovereignty:         sovereigntyLevel,
	}

	// Create via API
	result, err := r.client.CreateCarbonAwareSchedule(ctx, scheduleReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating carbon-aware schedule",
			fmt.Sprintf("Could not create carbon-aware schedule: %s", err),
		)
		return
	}

	// Map response to state
	r.mapToState(result, &plan, sovereigntyLevel)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "created carbon-aware schedule", map[string]interface{}{
		"id":          result.ID,
		"workload_id": result.WorkloadID,
		"sovereignty": result.Sovereignty,
	})
}

// Read reads the carbon-aware schedule state from the API for drift detection.
func (r *CarbonAwareScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CarbonAwareScheduleResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read from API
	result, err := r.client.GetCarbonAwareSchedule(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "carbon-aware schedule not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading carbon-aware schedule",
			fmt.Sprintf("Could not read carbon-aware schedule %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	// Map response to state
	r.mapToState(result, &state, result.Sovereignty)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the carbon-aware schedule resource.
func (r *CarbonAwareScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CarbonAwareScheduleResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine effective sovereignty
	sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sovereignty configuration", err.Error())
		return
	}

	// Build update request
	scheduleReq := &client.CarbonAwareSchedule{
		WorkloadID:          plan.WorkloadID.ValueString(),
		Enabled:             plan.Enabled.ValueBool(),
		MaxCarbonIntensity:  plan.MaxCarbonIntensity.ValueFloat64(),
		PreferredRegion:     plan.PreferredRegion.ValueString(),
		DeferWhenHighCarbon: plan.DeferWhenHighCarbon.ValueBool(),
		GreenWindowOnly:     plan.GreenWindowOnly.ValueBool(),
		Sovereignty:         sovereigntyLevel,
	}

	// Update via API
	result, err := r.client.UpdateCarbonAwareSchedule(ctx, state.ID.ValueString(), scheduleReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating carbon-aware schedule",
			fmt.Sprintf("Could not update carbon-aware schedule %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	// Map response to state
	r.mapToState(result, &plan, sovereigntyLevel)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "updated carbon-aware schedule", map[string]interface{}{
		"id":          result.ID,
		"workload_id": result.WorkloadID,
	})
}

// Delete deletes the carbon-aware schedule resource.
func (r *CarbonAwareScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CarbonAwareScheduleResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete via API
	err := r.client.DeleteCarbonAwareSchedule(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			// Already deleted
			tflog.Warn(ctx, "carbon-aware schedule already deleted", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting carbon-aware schedule",
			fmt.Sprintf("Could not delete carbon-aware schedule %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "deleted carbon-aware schedule", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

// ImportState imports an existing carbon-aware schedule into Terraform.
func (r *CarbonAwareScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapToState maps an API response to the Terraform state model.
func (r *CarbonAwareScheduleResource) mapToState(_ context.Context, schedule *client.CarbonAwareSchedule, model *CarbonAwareScheduleResourceModel, sovereigntyLevel string) {
	model.ID = types.StringValue(schedule.ID)
	model.WorkloadID = types.StringValue(schedule.WorkloadID)
	model.Enabled = types.BoolValue(schedule.Enabled)
	model.MaxCarbonIntensity = types.Float64Value(schedule.MaxCarbonIntensity)
	model.PreferredRegion = types.StringValue(schedule.PreferredRegion)
	model.DeferWhenHighCarbon = types.BoolValue(schedule.DeferWhenHighCarbon)
	model.GreenWindowOnly = types.BoolValue(schedule.GreenWindowOnly)
	model.Sovereignty = types.StringValue(sovereigntyLevel)
	model.Status = types.StringValue(schedule.Status)
	model.CurrentIntensity = types.Float64Value(schedule.CurrentIntensity)
	model.RecommendedAction = types.StringValue(schedule.RecommendedAction)
	model.NextGreenWindow = types.StringValue(schedule.NextGreenWindow)
	model.CreatedAt = types.StringValue(schedule.CreatedAt)
	model.UpdatedAt = types.StringValue(schedule.UpdatedAt)
}
