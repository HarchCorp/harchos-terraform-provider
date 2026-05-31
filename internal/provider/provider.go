package provider

import (
        "context"
        "fmt"
        "os"

        "github.com/HarchCorp/harchos-terraform-provider/internal/client"
        "github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
        "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
        "github.com/hashicorp/terraform-plugin-framework/datasource"
        "github.com/hashicorp/terraform-plugin-framework/path"
        "github.com/hashicorp/terraform-plugin-framework/provider"
        "github.com/hashicorp/terraform-plugin-framework/provider/schema"
        "github.com/hashicorp/terraform-plugin-framework/resource"
        "github.com/hashicorp/terraform-plugin-framework/schema/validator"
        "github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
        _ provider.Provider = &HarchOSProvider{}
)

// HarchOSProvider defines the provider implementation.
type HarchOSProvider struct {
        // version is set to the provider version on release, "dev" when the
        // provider is built and ran locally, and "test" when running acceptance
        // testing.
        version string
}

// HarchOSProviderModel describes the provider data model.
type HarchOSProviderModel struct {
        APIKey      types.String `tfsdk:"api_key"`
        Region      types.String `tfsdk:"region"`
        Sovereignty types.String `tfsdk:"sovereignty"`
        BaseURL     types.String `tfsdk:"base_url"`
}

// Metadata returns the provider type name.
func (p *HarchOSProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
        resp.TypeName = "harchos"
        resp.Version = p.version
}

// Schema returns the provider schema.
func (p *HarchOSProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
        resp.Schema = schema.Schema{
                MarkdownDescription: `
The HarchOS provider is used to interact with resources supported by
HarchCorp's HarchOS infrastructure platform. The provider needs to be
configured with the proper credentials before it can be used.

Use the navigation on the left to read about the available resources
and data sources.

## Sovereignty Enforcement

HarchOS enforces a strict data sovereignty model. Resources can be
assigned one of three sovereignty levels: **strict**, **regional**, or
**global**. Once a resource is created at a given level, it cannot be
downgraded to a less restrictive level (strict > regional > global).
The provider-level sovereignty acts as a default and a floor — resource-
level sovereignty will always be the more restrictive of the two.
`,
                Attributes: map[string]schema.Attribute{
                        "api_key": schema.StringAttribute{
                                MarkdownDescription: "The API key for accessing the HarchOS API. " +
                                        "May also be provided via HARCHOS_API_KEY environment variable.",
                                Optional:  true,
                                Sensitive: true,
                                Validators: []validator.String{
                                        stringvalidator.LengthAtLeast(1),
                                },
                        },
                        "region": schema.StringAttribute{
                                MarkdownDescription: "The default HarchOS region for resource deployment. " +
                                        "May also be provided via HARCHOS_REGION environment variable. " +
                                        "Common values: eu-west-1, us-east-1, ap-southeast-1.",
                                Optional: true,
                                Validators: []validator.String{
                                        stringvalidator.LengthAtLeast(1),
                                },
                        },
                        "sovereignty": schema.StringAttribute{
                                MarkdownDescription: "The default sovereignty level for resources. " +
                                        "Must be one of: strict, regional, global. " +
                                        "May also be provided via HARCHOS_SOVEREIGNTY environment variable. " +
                                        "Resource-level sovereignty overrides this if more restrictive.",
                                Optional: true,
                                Validators: []validator.String{
                                        sovereignty.SovereigntyLevelValidator(),
                                },
                        },
                        "base_url": schema.StringAttribute{
                                MarkdownDescription: "The base URL for the HarchOS API. " +
                                        "Defaults to https://api.harchos.ai/v1. " +
                                        "May also be provided via HARCHOS_BASE_URL environment variable. " +
                                        "Use this for testing against a local or staging API.",
                                Optional: true,
                                Validators: []validator.String{
                                        stringvalidator.LengthAtLeast(1),
                                },
                        },
                },
        }
}

// Configure configures the provider by reading configuration values and
// initializing the HarchOS API client.
func (p *HarchOSProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
        var config HarchOSProviderModel

        diags := req.Config.Get(ctx, &config)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        // If practitioner provided a null value for api_key, fall back to
        // environment variable.
        if config.APIKey.IsNull() {
                config.APIKey = types.StringValue(os.Getenv("HARCHOS_API_KEY"))
        }

        if config.Region.IsNull() {
                config.Region = types.StringValue(os.Getenv("HARCHOS_REGION"))
        }

        if config.Sovereignty.IsNull() {
                config.Sovereignty = types.StringValue(os.Getenv("HARCHOS_SOVEREIGNTY"))
        }

        if config.BaseURL.IsNull() {
                config.BaseURL = types.StringValue(os.Getenv("HARCHOS_BASE_URL"))
        }

        // Validate required configuration
        if config.APIKey.IsNull() || config.APIKey.ValueString() == "" {
                resp.Diagnostics.AddAttributeError(
                        path.Root("api_key"),
                        "Missing HarchOS API Key",
                        "The provider cannot create the HarchOS API client as there is no API key configured. "+
                                "Either set the api_key attribute in the provider configuration or set the "+
                                "HARCHOS_API_KEY environment variable.",
                )
        }

        if config.Region.IsNull() || config.Region.ValueString() == "" {
                resp.Diagnostics.AddAttributeError(
                        path.Root("region"),
                        "Missing HarchOS Region",
                        "The provider cannot create the HarchOS API client as there is no region configured. "+
                                "Either set the region attribute in the provider configuration or set the "+
                                "HARCHOS_REGION environment variable.",
                )
        }

        if resp.Diagnostics.HasError() {
                return
        }

        // Determine effective sovereignty
        sovereigntyLevel := ""
        if !config.Sovereignty.IsNull() {
                sovereigntyLevel = config.Sovereignty.ValueString()
                if !sovereignty.IsValid(sovereigntyLevel) {
                        resp.Diagnostics.AddAttributeError(
                                path.Root("sovereignty"),
                                "Invalid sovereignty level",
                                fmt.Sprintf("Expected one of %v, got %q", sovereignty.ValidLevels(), sovereigntyLevel),
                        )
                        return
                }
        }

        // Create the HarchOS API client
        clientCfg := client.Config{
                APIKey:      config.APIKey.ValueString(),
                Region:      config.Region.ValueString(),
                Sovereignty: sovereigntyLevel,
                BaseURL:     config.BaseURL.ValueString(),
                Version:     p.version,
        }

        apiClient, err := client.New(clientCfg)
        if err != nil {
                resp.Diagnostics.AddError(
                        "Failed to create HarchOS API client",
                        fmt.Sprintf("Could not initialize the HarchOS API client: %s", err),
                )
                return
        }

        // Make the client available to resources and data sources
        resp.DataSourceData = &ProviderData{
                Client:      apiClient,
                Region:      config.Region.ValueString(),
                Sovereignty: sovereigntyLevel,
        }
        resp.ResourceData = &ProviderData{
                Client:      apiClient,
                Region:      config.Region.ValueString(),
                Sovereignty: sovereigntyLevel,
        }
}

// Resources returns the list of resources supported by this provider.
func (p *HarchOSProvider) Resources(_ context.Context) []func() resource.Resource {
        return []func() resource.Resource{
                NewWorkloadResource,
                NewModelResource,
                NewInferenceEndpointResource,
                NewDatasetResource,
                NewNetworkPolicyResource,
                NewStorageVolumeResource,
                NewCarbonAwareScheduleResource,
        }
}

// DataSources returns the list of data sources supported by this provider.
func (p *HarchOSProvider) DataSources(_ context.Context) []func() datasource.DataSource {
        return []func() datasource.DataSource{
                NewHubsDataSource,
                NewWorkloadDataSource,
                NewModelDataSource,
                NewRegionsDataSource,
        }
}

// ProviderData wraps the API client and provider-level configuration
// that is shared with all resources and data sources.
type ProviderData struct {
        Client      *client.Client
        Region      string
        Sovereignty string
}

// NewHarchOSProvider creates a new provider instance with the given version.
func NewHarchOSProvider(version string) func() provider.Provider {
        return func() provider.Provider {
                return &HarchOSProvider{
                        version: version,
                }
        }
}
