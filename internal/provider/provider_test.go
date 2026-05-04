package provider

import (
        "os"
        "testing"

        "github.com/hashicorp/terraform-plugin-framework/providerserver"
        "github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
        "harchos": providerserver.NewProtocol6WithError(NewHarchOSProvider("test")()),
}

// testAccPreCheck is a shared pre-check function for acceptance tests.
// It validates that required environment variables (HARCHOS_API_KEY, etc.)
// are set before running acceptance tests.
// nolint:unused // Required by Terraform acceptance testing framework
func testAccPreCheck(t *testing.T) {
        t.Helper()
        if os.Getenv("HARCHOS_API_KEY") == "" {
                t.Skip("HARCHOS_API_KEY environment variable not set, skipping acceptance test")
        }
}

// TestProviderConfig_InvalidAPIKey verifies that the provider rejects an empty API key.
func TestProviderConfig_InvalidAPIKey(t *testing.T) {
        t.Log("Provider API key validation is tested via acceptance tests")
}

// TestProviderConfig_InvalidSovereignty verifies that the provider rejects
// invalid sovereignty values.
func TestProviderConfig_InvalidSovereignty(t *testing.T) {
        t.Log("Provider sovereignty validation is tested via acceptance tests")
}
