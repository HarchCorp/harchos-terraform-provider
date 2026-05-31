package provider

import (
        "context"
        "fmt"

        "github.com/HarchCorp/harchos-terraform-provider/internal/client"
        "github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
        "github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
        "github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
        "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
        "github.com/hashicorp/terraform-plugin-framework/attr"
        "github.com/hashicorp/terraform-plugin-framework/diag"
        "github.com/hashicorp/terraform-plugin-framework/path"
        "github.com/hashicorp/terraform-plugin-framework/resource"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
        "github.com/hashicorp/terraform-plugin-framework/schema/validator"
        "github.com/hashicorp/terraform-plugin-framework/types"
        "github.com/hashicorp/terraform-plugin-framework/types/basetypes"
        "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
        _ resource.Resource                = &WorkloadResource{}
        _ resource.ResourceWithConfigure   = &WorkloadResource{}
        _ resource.ResourceWithImportState = &WorkloadResource{}
        _ resource.ResourceWithModifyPlan  = &WorkloadResource{}
)

// WorkloadResource defines the resource implementation.
type WorkloadResource struct {
        client      *client.Client
        region      string
        sovereignty string
}

// WorkloadResourceModel describes the resource data model.
type WorkloadResourceModel struct {
        ID          types.String `tfsdk:"id"`
        Name        types.String `tfsdk:"name"`
        Image       types.String `tfsdk:"image"`
        Replicas    types.Int64  `tfsdk:"replicas"`
        Region      types.String `tfsdk:"region"`
        Sovereignty types.String `tfsdk:"sovereignty"`
        Env         types.Map    `tfsdk:"env"`
        Tags        types.Map    `tfsdk:"tags"`
        Status      types.String `tfsdk:"status"`
        CreatedAt   types.String `tfsdk:"created_at"`
        UpdatedAt   types.String `tfsdk:"updated_at"`
}

// NewWorkloadResource returns a new workload resource.
func NewWorkloadResource() resource.Resource {
        return &WorkloadResource{}
}

// Metadata returns the resource type name.
func (r *WorkloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
        resp.TypeName = req.ProviderTypeName + "_workload"
}

// Schema returns the resource schema.
func (r *WorkloadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
        resp.Schema = schema.Schema{
                MarkdownDescription: "Manages a HarchOS workload. Workloads are the primary compute unit " +
                        "in HarchOS, running containerized workloads across sovereign hubs.",

                Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                                MarkdownDescription: "The unique identifier of the workload.",
                                Computed:            true,
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.UseStateForUnknown(),
                                },
                        },
                        "name": schema.StringAttribute{
                                MarkdownDescription: "The name of the workload. Must be unique within the region.",
                                Required:            true,
                                Validators: []validator.String{
                                        stringvalidator.LengthBetween(1, 128),
                                },
                        },
                        "image": schema.StringAttribute{
                                MarkdownDescription: "The container image to deploy (e.g., nginx:latest, harchos/ml-inference:v2). Changing this field requires resource replacement.",
                                Required:            true,
                                Validators: []validator.String{
                                        stringvalidator.LengthAtLeast(1),
                                },
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.RequiresReplace(),
                                },
                        },
                        "replicas": schema.Int64Attribute{
                                MarkdownDescription: "The number of workload replicas. Defaults to 1.",
                                Optional:            true,
                                Computed:            true,
                                Default:             int64default.StaticInt64(1),
                                Validators: []validator.Int64{
                                        int64validator.AtLeast(1),
                                        int64validator.AtMost(100),
                                },
                        },
                        "region": schema.StringAttribute{
                                MarkdownDescription: "The HarchOS region where the workload is deployed. " +
                                        "Defaults to the provider region.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                                Validators: []validator.String{
                                        stringvalidator.LengthAtLeast(1),
                                },
                        },
                        "sovereignty": schema.StringAttribute{
                                MarkdownDescription: "The sovereignty level for this workload. Must be one of: " +
                                        "strict, regional, global. Cannot be downgraded after creation. " +
                                        "Defaults to the provider sovereignty if not set.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                                Validators: []validator.String{
                                        sovereignty.SovereigntyLevelValidator(),
                                },
                        },
                        "env": schema.MapAttribute{
                                MarkdownDescription: "Environment variables to inject into the workload containers.",
                                Optional:            true,
                                Computed:            true,
                                Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
                                ElementType:         types.StringType,
                                Validators: []validator.Map{
                                        mapvalidator.SizeAtMost(64),
                                },
                        },
                        "tags": schema.MapAttribute{
                                MarkdownDescription: "Key-value tags for resource organization and billing.",
                                Optional:            true,
                                Computed:            true,
                                Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
                                ElementType:         types.StringType,
                                Validators: []validator.Map{
                                        mapvalidator.SizeAtMost(32),
                                },
                        },
                        "status": schema.StringAttribute{
                                MarkdownDescription: "The current status of the workload.",
                                Computed:            true,
                        },
                        "created_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the workload was created.",
                                Computed:            true,
                        },
                        "updated_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the workload was last updated.",
                                Computed:            true,
                        },
                },
        }
}

// Configure adds the provider-configured client to the resource.
func (r *WorkloadResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
        if req.ProviderData == nil {
                return
        }

        providerData, ok := req.ProviderData.(*ProviderData)
        if !ok {
                resp.Diagnostics.AddError(
                        "Unexpected Resource Configure Type",
                        fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
                )
                return
        }

        r.client = providerData.Client
        r.region = providerData.Region
        r.sovereignty = providerData.Sovereignty
}

// ModifyPlan enables drift detection and sovereignty enforcement at plan time.
func (r *WorkloadResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
        var plan, state WorkloadResourceModel

        // If there's no plan, there's nothing to modify
        if req.Plan.Raw.IsNull() {
                return
        }

        diags := req.Plan.Get(ctx, &plan)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        // Default region from provider if not specified
        if plan.Region.IsNull() || plan.Region.ValueString() == "" {
                resp.Plan.SetAttribute(ctx, path.Root("region"), r.region)
        }

        // Default sovereignty from provider if not specified
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
                                fmt.Sprintf("Cannot change workload sovereignty: %s", err),
                        )
                        return
                }
        }

        // Compute effective sovereignty for compliance
        effectiveSovereignty, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
        if err != nil {
                resp.Diagnostics.AddWarning(
                        "Sovereignty calculation warning",
                        err.Error(),
                )
        } else {
                tflog.Debug(ctx, "workload effective sovereignty", map[string]interface{}{
                        "sovereignty": effectiveSovereignty,
                })
        }
}

// Create creates the workload resource.
func (r *WorkloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
        var plan WorkloadResourceModel

        diags := req.Plan.Get(ctx, &plan)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        // Determine region
        region := r.region
        if !plan.Region.IsNull() && plan.Region.ValueString() != "" {
                region = plan.Region.ValueString()
        }

        // Determine effective sovereignty
        sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
        if err != nil {
                resp.Diagnostics.AddError(
                        "Invalid sovereignty configuration",
                        err.Error(),
                )
                return
        }

        // Build the API request
        workloadReq := &client.Workload{
                Name:        plan.Name.ValueString(),
                Image:       plan.Image.ValueString(),
                Replicas:    int(plan.Replicas.ValueInt64()),
                Region:      region,
                Sovereignty: sovereigntyLevel,
                Env:         stringMapFromFramework(ctx, plan.Env, &resp.Diagnostics),
                Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        // Create the workload via API
        result, err := r.client.CreateWorkload(ctx, workloadReq)
        if err != nil {
                resp.Diagnostics.AddError(
                        "Error creating workload",
                        fmt.Sprintf("Could not create workload: %s", err),
                )
                return
        }

        // Map response to state
        r.mapToState(ctx, result, &plan, region, sovereigntyLevel)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)

        tflog.Info(ctx, "created workload", map[string]interface{}{
                "id":          result.ID,
                "name":        result.Name,
                "sovereignty": result.Sovereignty,
        })
}

// Read reads the workload state from the API for drift detection.
func (r *WorkloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
        var state WorkloadResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        // Read the workload from API
        result, err := r.client.GetWorkload(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        tflog.Warn(ctx, "workload not found, removing from state", map[string]interface{}{
                                "id": state.ID.ValueString(),
                        })
                        resp.State.RemoveResource(ctx)
                        return
                }
                resp.Diagnostics.AddError(
                        "Error reading workload",
                        fmt.Sprintf("Could not read workload %s: %s", state.ID.ValueString(), err),
                )
                return
        }

        // Map response to state
        r.mapToState(ctx, result, &state, result.Region, result.Sovereignty)

        diags = resp.State.Set(ctx, state)
        resp.Diagnostics.Append(diags...)
}

// Update updates the workload resource.
func (r *WorkloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
        var plan, state WorkloadResourceModel

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

        // Determine region
        region := r.region
        if !plan.Region.IsNull() && plan.Region.ValueString() != "" {
                region = plan.Region.ValueString()
        }

        // Determine effective sovereignty
        sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
        if err != nil {
                resp.Diagnostics.AddError(
                        "Invalid sovereignty configuration",
                        err.Error(),
                )
                return
        }

        // Build update request
        workloadReq := &client.Workload{
                Name:        plan.Name.ValueString(),
                Image:       plan.Image.ValueString(),
                Replicas:    int(plan.Replicas.ValueInt64()),
                Region:      region,
                Sovereignty: sovereigntyLevel,
                Env:         stringMapFromFramework(ctx, plan.Env, &resp.Diagnostics),
                Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        // Update via API
        result, err := r.client.UpdateWorkload(ctx, state.ID.ValueString(), workloadReq)
        if err != nil {
                resp.Diagnostics.AddError(
                        "Error updating workload",
                        fmt.Sprintf("Could not update workload %s: %s", state.ID.ValueString(), err),
                )
                return
        }

        // Map response to state
        r.mapToState(ctx, result, &plan, region, sovereigntyLevel)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)

        tflog.Info(ctx, "updated workload", map[string]interface{}{
                "id":   result.ID,
                "name": result.Name,
        })
}

// Delete deletes the workload resource.
func (r *WorkloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
        var state WorkloadResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        // Delete via API
        err := r.client.DeleteWorkload(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        // Already deleted
                        tflog.Warn(ctx, "workload already deleted", map[string]interface{}{
                                "id": state.ID.ValueString(),
                        })
                        return
                }
                resp.Diagnostics.AddError(
                        "Error deleting workload",
                        fmt.Sprintf("Could not delete workload %s: %s", state.ID.ValueString(), err),
                )
                return
        }

        tflog.Info(ctx, "deleted workload", map[string]interface{}{
                "id": state.ID.ValueString(),
        })
}

// ImportState imports an existing workload into Terraform.
func (r *WorkloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
        resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapToState maps an API response to the Terraform state model.
func (r *WorkloadResource) mapToState(ctx context.Context, workload *client.Workload, model *WorkloadResourceModel, region, sovereigntyLevel string) {
        model.ID = types.StringValue(workload.ID)
        model.Name = types.StringValue(workload.Name)
        model.Image = types.StringValue(workload.Image)
        model.Replicas = types.Int64Value(int64(workload.Replicas))
        model.Region = types.StringValue(region)
        model.Sovereignty = types.StringValue(sovereigntyLevel)
        model.Status = types.StringValue(workload.Status)
        model.CreatedAt = types.StringValue(workload.CreatedAt)
        model.UpdatedAt = types.StringValue(workload.UpdatedAt)

        model.Env = frameworkFromStringMap(ctx, workload.Env)
        model.Tags = frameworkFromStringMap(ctx, workload.Tags)
}

// stringMapFromFramework converts a Framework map to a Go map[string]string.
func stringMapFromFramework(ctx context.Context, m types.Map, diags *diag.Diagnostics) map[string]string {
        if m.IsNull() || m.IsUnknown() {
                return map[string]string{}
        }

        result := make(map[string]string, len(m.Elements()))
        for k, v := range m.Elements() {
                s, ok := v.(basetypes.StringValue)
                if !ok {
                        diags.AddError(
                                "Type conversion error",
                                fmt.Sprintf("Expected string value for key %q, got %T", k, v),
                        )
                        continue
                }
                result[k] = s.ValueString()
        }

        return result
}

// frameworkFromStringMap converts a Go map[string]string to a Framework types.Map.
func frameworkFromStringMap(ctx context.Context, m map[string]string) types.Map {
        if m == nil {
                return types.MapNull(types.StringType)
        }

        elements := make(map[string]attr.Value, len(m))
        for k, v := range m {
                elements[k] = types.StringValue(v)
        }

        result, diags := types.MapValue(types.StringType, elements)
        if diags.HasError() {
                return types.MapNull(types.StringType)
        }

        return result
}
