package provider

import (
	"fmt"
	"testing"

	"github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccWorkloadResource tests the full lifecycle of a harchos_workload resource.
func TestAccWorkloadResource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWorkloadResourceConfig("test-workload", "nginx:latest", 2, "strict"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("harchos_workload.test", "name", "test-workload"),
					resource.TestCheckResourceAttr("harchos_workload.test", "image", "nginx:latest"),
					resource.TestCheckResourceAttr("harchos_workload.test", "replicas", "2"),
					resource.TestCheckResourceAttr("harchos_workload.test", "sovereignty", "strict"),
					// Verify computed attributes are set
					resource.TestCheckResourceAttrSet("harchos_workload.test", "id"),
					resource.TestCheckResourceAttrSet("harchos_workload.test", "status"),
					resource.TestCheckResourceAttrSet("harchos_workload.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "harchos_workload.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Computed attributes may differ on import
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			// Update with increased replicas
			{
				Config: testAccWorkloadResourceConfig("test-workload", "nginx:latest", 3, "strict"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("harchos_workload.test", "replicas", "3"),
				),
			},
		},
	})
}

// TestAccWorkloadResource_SovereigntyDowngrade tests that sovereignty
// downgrades are rejected at plan time.
func TestAccWorkloadResource_SovereigntyDowngrade(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with strict sovereignty
			{
				Config: testAccWorkloadResourceConfig("sovereignty-test", "nginx:latest", 1, "strict"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("harchos_workload.test", "sovereignty", "strict"),
				),
			},
			// Attempt to downgrade to regional - should fail
			{
				Config:      testAccWorkloadResourceConfig("sovereignty-test", "nginx:latest", 1, "regional"),
				ExpectError: sovereigntyDowngradeError(),
			},
		},
	})
}

// TestAccWorkloadDataSource tests reading a workload via data source.
func TestAccWorkloadDataSource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkloadDataSourceConfig("ds-test-workload", "nginx:latest", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.harchos_workload.test", "name", "ds-test-workload"),
					resource.TestCheckResourceAttr("data.harchos_workload.test", "image", "nginx:latest"),
					resource.TestCheckResourceAttrSet("data.harchos_workload.test", "id"),
				),
			},
		},
	})
}

// Unit tests for sovereignty validation

func TestSovereigntyValidation_ValidLevels(t *testing.T) {
	tests := []struct {
		level string
		valid bool
	}{
		{"strict", true},
		{"regional", true},
		{"global", true},
		{"invalid", false},
		{"", false},
		{"STRICT", true}, // case insensitive
		{"Regional", true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if sovereignty.IsValid(tt.level) != tt.valid {
				t.Errorf("IsValid(%q) = %v, want %v", tt.level, !tt.valid, tt.valid)
			}
		})
	}
}

func TestSovereigntyTransition_DowngradePrevention(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		proposed string
		allowed  bool
	}{
		{"strict to strict", "strict", "strict", true},
		{"strict to regional", "strict", "regional", false},
		{"strict to global", "strict", "global", false},
		{"regional to strict", "regional", "strict", true},
		{"regional to regional", "regional", "regional", true},
		{"regional to global", "regional", "global", false},
		{"global to strict", "global", "strict", true},
		{"global to regional", "global", "regional", true},
		{"global to global", "global", "global", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sovereignty.CanTransition(tt.current, tt.proposed)
			if tt.allowed && err != nil {
				t.Errorf("CanTransition(%q, %q) returned unexpected error: %v", tt.current, tt.proposed, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("CanTransition(%q, %q) should have returned an error", tt.current, tt.proposed)
			}
		})
	}
}

func TestEffectiveSovereignty(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		resource  string
		expected  string
		wantError bool
	}{
		{"provider strict, resource empty", "strict", "", "strict", false},
		{"provider empty, resource regional", "", "regional", "regional", false},
		{"provider regional, resource strict", "regional", "strict", "strict", false},
		{"provider strict, resource regional", "strict", "regional", "strict", false},
		{"provider global, resource regional", "global", "regional", "regional", false},
		{"both empty", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sovereignty.EffectiveSovereignty(tt.provider, tt.resource)
			if tt.wantError && err == nil {
				t.Errorf("EffectiveSovereignty(%q, %q) should have returned an error", tt.provider, tt.resource)
			}
			if !tt.wantError && err != nil {
				t.Errorf("EffectiveSovereignty(%q, %q) returned unexpected error: %v", tt.provider, tt.resource, err)
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("EffectiveSovereignty(%q, %q) = %q, want %q", tt.provider, tt.resource, result, tt.expected)
			}
		})
	}
}

// Helper functions for test configurations

func testAccWorkloadResourceConfig(name, image string, replicas int, sovereigntyLevel string) string {
	return fmt.Sprintf(`
resource "harchos_workload" "test" {
  name        = %[1]q
  image       = %[2]q
  replicas    = %[3]d
  sovereignty = %[4]q
  region      = "eu-west-1"

  env = {
    LOG_LEVEL = "info"
  }

  tags = {
    Environment = "test"
    ManagedBy   = "terraform"
  }
}
`, name, image, replicas, sovereigntyLevel)
}

func testAccWorkloadDataSourceConfig(name, image string, replicas int) string {
	return fmt.Sprintf(`
resource "harchos_workload" "test" {
  name     = %[1]q
  image    = %[2]q
  replicas = %[3]d
  region   = "eu-west-1"
}

data "harchos_workload" "test" {
  id = harchos_workload.test.id
}
`, name, image, replicas)
}

func sovereigntyDowngradeError() string {
	return "sovereignty cannot be downgraded"
}
