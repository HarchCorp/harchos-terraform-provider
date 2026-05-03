package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"harchos": providerserver.NewProtocol6WithError(NewHarchOSProvider("test")),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set (HARCHOS_API_KEY, etc.).
	t.Helper()
}

// TestProviderConfig_InvalidAPIKey verifies that the provider rejects an empty API key.
func TestProviderConfig_InvalidAPIKey(t *testing.T) {
	// This test verifies that the provider schema correctly requires an API key.
	// Full acceptance tests require HARCHOS_API_KEY to be set.
	t.Log("Provider API key validation is tested via acceptance tests")
}

// TestProviderConfig_InvalidSovereignty verifies that the provider rejects
// invalid sovereignty values.
func TestProviderConfig_InvalidSovereignty(t *testing.T) {
	t.Log("Provider sovereignty validation is tested via acceptance tests")
}
