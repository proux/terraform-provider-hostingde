package hostingde

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultBaseURL = "https://secure.hosting.de/api/dns/v1/json"

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &hostingdeProvider{}
)

// hostingdeProviderModel maps provider schema data to a Go type.
type hostingdeProviderModel struct {
	AccountId types.String `tfsdk:"account_id"`
	AuthToken types.String `tfsdk:"auth_token"`
	BaseUrl   types.String `tfsdk:"base_url"`
}

// New is a helper function to simplify provider server and testing implementation.
func New() provider.Provider {
	return &hostingdeProvider{}
}

// hostingdeProvider is the provider implementation.
type hostingdeProvider struct{}

// Metadata returns the provider type name.
func (p *hostingdeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hostingde"
}

// Schema defines the provider-level schema for configuration data.
func (p *hostingdeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Description: "Account ID for hosting.de API. May also be provided via HOSTINGDE_ACCOUNT_ID environment variable.",
				Optional:    true,
			},
			"auth_token": schema.StringAttribute{
				Description: "Auth token for hosting.de API. May also be provided via HOSTINGDE_AUTH_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Base URL for hosting.de API. May also be provided via HOSTINGDE_BASE_URL environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *hostingdeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring hosting.de client")
	// Retrieve provider data from configuration
	var config hostingdeProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if config.AccountId.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Unknown hosting.de API account ID",
			"The provider cannot create the hosting.de API client as there is an unknown configuration value for the hosting.de API account ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the HOSTINGDE_ACCOUNT_ID environment variable.",
		)
	}

	if config.AuthToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_token"),
			"Unknown hosting.de API auth token",
			"The provider cannot create the hosting.de API client as there is an unknown configuration value for the hosting.de API auth token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the HOSTINGDE_AUTH_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	account_id := os.Getenv("HOSTINGDE_ACCOUNT_ID")
	auth_token := os.Getenv("HOSTINGDE_AUTH_TOKEN")
	base_url := os.Getenv("HOSTINGDE_BASE_URL")

	if !config.AccountId.IsNull() {
		account_id = config.AccountId.ValueString()
	}

	if !config.AuthToken.IsNull() {
		auth_token = config.AuthToken.ValueString()
	}

	if !config.BaseUrl.IsNull() {
		base_url = config.BaseUrl.ValueString()
	}

	// Default for API Base URL
	if base_url == "" {
		base_url = defaultBaseURL
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if auth_token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_token"),
			"Missing hosting.de API auth token",
			"The provider cannot create the hosting.de API client as there is a missing or empty value for the hosting.de API auth token. "+
				"Set the auth_token value in the configuration or use the HOSTINGDE_AUTH_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "hostingde_account_id", account_id)
	ctx = tflog.SetField(ctx, "hostingde_auth_token", auth_token)
	ctx = tflog.SetField(ctx, "hostingde_base_url", base_url)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "hostingde_auth_token")

	tflog.Debug(ctx, "Creating hosting.de client")

	// Create a new hosting.de client using the configuration values
	client := NewClient(&account_id, &auth_token, &base_url)

	// Make the hosting.de client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured hosting.de client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *hostingdeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *hostingdeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZoneResource,
		NewRecordResource,
	}
}
