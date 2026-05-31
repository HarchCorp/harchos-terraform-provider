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
        "github.com/hashicorp/terraform-plugin-framework/path"
        "github.com/hashicorp/terraform-plugin-framework/resource"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
        "github.com/hashicorp/terraform-plugin-framework/schema/validator"
        "github.com/hashicorp/terraform-plugin-framework/types"
        "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
        _ resource.Resource                = &StorageVolumeResource{}
        _ resource.ResourceWithConfigure   = &StorageVolumeResource{}
        _ resource.ResourceWithImportState = &StorageVolumeResource{}
        _ resource.ResourceWithModifyPlan  = &StorageVolumeResource{}
)

// StorageVolumeResource defines the resource implementation.
type StorageVolumeResource struct {
        client      *client.Client
        region      string
        sovereignty string
}

// StorageVolumeResourceModel describes the resource data model.
type StorageVolumeResourceModel struct {
        ID          types.String `tfsdk:"id"`
        Name        types.String `tfsdk:"name"`
        SizeGB      types.Int64  `tfsdk:"size_gb"`
        VolumeType  types.String `tfsdk:"volume_type"`
        Region      types.String `tfsdk:"region"`
        Sovereignty types.String `tfsdk:"sovereignty"`
        Encrypted   types.Bool   `tfsdk:"encrypted"`
        Tags        types.Map    `tfsdk:"tags"`
        Status      types.String `tfsdk:"status"`
        CreatedAt   types.String `tfsdk:"created_at"`
        UpdatedAt   types.String `tfsdk:"updated_at"`
}

// NewStorageVolumeResource returns a new storage volume resource.
func NewStorageVolumeResource() resource.Resource {
        return &StorageVolumeResource{}
}

// Metadata returns the resource type name.
func (r *StorageVolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
        resp.TypeName = req.ProviderTypeName + "_storage_volume"
}

// Schema returns the resource schema.
func (r *StorageVolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
        resp.Schema = schema.Schema{
                MarkdownDescription: "Manages a HarchOS storage volume. Storage volumes provide " +
                        "persistent, sovereign block storage for workloads and datasets with " +
                        "configurable encryption and volume types.",

                Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                                MarkdownDescription: "The unique identifier of the storage volume.",
                                Computed:            true,
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.UseStateForUnknown(),
                                },
                        },
                        "name": schema.StringAttribute{
                                MarkdownDescription: "The name of the storage volume.",
                                Required:            true,
                                Validators: []validator.String{
                                        stringvalidator.LengthBetween(1, 128),
                                },
                        },
                        "size_gb": schema.Int64Attribute{
                                MarkdownDescription: "The size of the volume in gigabytes.",
                                Required:            true,
                                Validators: []validator.Int64{
                                        int64validator.AtLeast(1),
                                        int64validator.AtMost(16384),
                                },
                        },
                        "volume_type": schema.StringAttribute{
                                MarkdownDescription: "The volume type (ssd, hdd, nvme). Defaults to ssd. Changing this field requires resource replacement.",
                                Optional:            true,
                                Computed:            true,
                                Default:             stringdefault.StaticString("ssd"),
                                Validators: []validator.String{
                                        stringvalidator.OneOf("ssd", "hdd", "nvme"),
                                },
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.RequiresReplace(),
                                },
                        },
                        "region": schema.StringAttribute{
                                MarkdownDescription: "The HarchOS region for the volume. Defaults to provider region.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                        },
                        "sovereignty": schema.StringAttribute{
                                MarkdownDescription: "The sovereignty level for this volume. Strict sovereignty " +
                                        "volumes cannot be replicated across regions. Cannot be downgraded.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                                Validators: []validator.String{
                                        sovereignty.SovereigntyLevelValidator(),
                                },
                        },
                        "encrypted": schema.BoolAttribute{
                                MarkdownDescription: "Whether the volume is encrypted at rest. Defaults to true. Changing this field requires resource replacement.",
                                Optional:            true,
                                Computed:            true,
                                Default:             booldefault.StaticBool(true),
                                PlanModifiers: []planmodifier.Bool{
                                        boolplanmodifier.RequiresReplace(),
                                },
                        },
                        "tags": schema.MapAttribute{
                                MarkdownDescription: "Key-value tags for resource organization.",
                                Optional:            true,
                                Computed:            true,
                                Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
                                ElementType:         types.StringType,
                                Validators: []validator.Map{
                                        mapvalidator.SizeAtMost(32),
                                },
                        },
                        "status": schema.StringAttribute{
                                MarkdownDescription: "The current status of the storage volume.",
                                Computed:            true,
                        },
                        "created_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the volume was created.",
                                Computed:            true,
                        },
                        "updated_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the volume was last updated.",
                                Computed:            true,
                        },
                },
        }
}

// Configure adds the provider-configured client to the resource.
func (r *StorageVolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan enables drift detection and sovereignty enforcement.
func (r *StorageVolumeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
        var plan, state StorageVolumeResourceModel

        if req.Plan.Raw.IsNull() {
                return
        }

        diags := req.Plan.Get(ctx, &plan)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        if plan.Region.IsNull() || plan.Region.ValueString() == "" {
                resp.Plan.SetAttribute(ctx, path.Root("region"), r.region)
        }

        if plan.Sovereignty.IsNull() || plan.Sovereignty.ValueString() == "" {
                resp.Plan.SetAttribute(ctx, path.Root("sovereignty"), r.sovereignty)
        }

        if !req.State.Raw.IsNull() {
                diags = req.State.Get(ctx, &state)
                resp.Diagnostics.Append(diags...)
                if resp.Diagnostics.HasError() {
                        return
                }

                if err := sovereignty.ValidateSovereigntyTransition(state.Sovereignty, plan.Sovereignty); err != nil {
                        resp.Diagnostics.AddError(
                                "Sovereignty downgrade not allowed",
                                fmt.Sprintf("Cannot change storage volume sovereignty: %s", err),
                        )
                }

                // Volume shrinking is not supported
                if state.SizeGB.ValueInt64() > plan.SizeGB.ValueInt64() {
                        resp.Diagnostics.AddError(
                                "Volume shrink not supported",
                                fmt.Sprintf("Cannot shrink storage volume from %dGB to %dGB. HarchOS only supports volume expansion.",
                                        state.SizeGB.ValueInt64(), plan.SizeGB.ValueInt64()),
                        )
                }
        }
}

// Create creates the storage volume resource.
func (r *StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
        var plan StorageVolumeResourceModel

        diags := req.Plan.Get(ctx, &plan)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        region := r.region
        if !plan.Region.IsNull() && plan.Region.ValueString() != "" {
                region = plan.Region.ValueString()
        }

        sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
        if err != nil {
                resp.Diagnostics.AddError("Invalid sovereignty configuration", err.Error())
                return
        }

        volumeReq := &client.StorageVolume{
                Name:        plan.Name.ValueString(),
                SizeGB:      int(plan.SizeGB.ValueInt64()),
                VolumeType:  plan.VolumeType.ValueString(),
                Region:      region,
                Sovereignty: sovereigntyLevel,
                Encrypted:   plan.Encrypted.ValueBool(),
                Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.CreateStorageVolume(ctx, volumeReq)
        if err != nil {
                resp.Diagnostics.AddError("Error creating storage volume",
                        fmt.Sprintf("Could not create storage volume: %s", err))
                return
        }

        plan.ID = types.StringValue(result.ID)
        plan.Region = types.StringValue(region)
        plan.Sovereignty = types.StringValue(sovereigntyLevel)
        plan.Encrypted = types.BoolValue(result.Encrypted)
        plan.Status = types.StringValue(result.Status)
        plan.CreatedAt = types.StringValue(result.CreatedAt)
        plan.UpdatedAt = types.StringValue(result.UpdatedAt)
        plan.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)

        tflog.Info(ctx, "created storage volume", map[string]interface{}{
                "id": result.ID, "name": result.Name, "size_gb": result.SizeGB,
        })
}

// Read reads the storage volume state from the API.
func (r *StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
        var state StorageVolumeResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.GetStorageVolume(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        resp.State.RemoveResource(ctx)
                        return
                }
                resp.Diagnostics.AddError("Error reading storage volume",
                        fmt.Sprintf("Could not read storage volume %s: %s", state.ID.ValueString(), err))
                return
        }

        state.ID = types.StringValue(result.ID)
        state.Name = types.StringValue(result.Name)
        state.SizeGB = types.Int64Value(int64(result.SizeGB))
        state.VolumeType = types.StringValue(result.VolumeType)
        state.Region = types.StringValue(result.Region)
        state.Sovereignty = types.StringValue(result.Sovereignty)
        state.Encrypted = types.BoolValue(result.Encrypted)
        state.Status = types.StringValue(result.Status)
        state.CreatedAt = types.StringValue(result.CreatedAt)
        state.UpdatedAt = types.StringValue(result.UpdatedAt)
        state.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, state)
        resp.Diagnostics.Append(diags...)
}

// Update updates the storage volume resource.
func (r *StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
        var plan, state StorageVolumeResourceModel

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

        region := r.region
        if !plan.Region.IsNull() && plan.Region.ValueString() != "" {
                region = plan.Region.ValueString()
        }

        sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
        if err != nil {
                resp.Diagnostics.AddError("Invalid sovereignty configuration", err.Error())
                return
        }

        volumeReq := &client.StorageVolume{
                Name:        plan.Name.ValueString(),
                SizeGB:      int(plan.SizeGB.ValueInt64()),
                VolumeType:  plan.VolumeType.ValueString(),
                Region:      region,
                Sovereignty: sovereigntyLevel,
                Encrypted:   plan.Encrypted.ValueBool(),
                Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.UpdateStorageVolume(ctx, state.ID.ValueString(), volumeReq)
        if err != nil {
                resp.Diagnostics.AddError("Error updating storage volume",
                        fmt.Sprintf("Could not update storage volume %s: %s", state.ID.ValueString(), err))
                return
        }

        plan.ID = types.StringValue(result.ID)
        plan.Region = types.StringValue(region)
        plan.Sovereignty = types.StringValue(sovereigntyLevel)
        plan.Encrypted = types.BoolValue(result.Encrypted)
        plan.Status = types.StringValue(result.Status)
        plan.CreatedAt = types.StringValue(result.CreatedAt)
        plan.UpdatedAt = types.StringValue(result.UpdatedAt)
        plan.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)
}

// Delete deletes the storage volume resource.
func (r *StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
        var state StorageVolumeResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        err := r.client.DeleteStorageVolume(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        return
                }
                resp.Diagnostics.AddError("Error deleting storage volume",
                        fmt.Sprintf("Could not delete storage volume %s: %s", state.ID.ValueString(), err))
                return
        }

        tflog.Info(ctx, "deleted storage volume", map[string]interface{}{"id": state.ID.ValueString()})
}

// ImportState imports an existing storage volume into Terraform.
func (r *StorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
        resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
