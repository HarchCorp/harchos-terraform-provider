package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Region represents a HarchOS region with metadata.
type Region struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Country              string  `json:"country"`
	Continent            string  `json:"continent"`
	CarbonIntensity      float64 `json:"carbon_intensity"`
	RenewablePercentage  float64 `json:"renewable_percentage"`
	Latitude             float64 `json:"latitude,omitempty"`
	Longitude            float64 `json:"longitude,omitempty"`
	SovereigntyDefault   string  `json:"sovereignty_default,omitempty"`
	AvailableGPUModels   []string `json:"available_gpu_models,omitempty"`
	GPUCount             int     `json:"gpu_count,omitempty"`
	ActiveWorkloads      int     `json:"active_workloads,omitempty"`
	PUE                  float64 `json:"pue,omitempty"`
	GreenEnergySource    string  `json:"green_energy_source,omitempty"`
	PricingTier          string  `json:"pricing_tier,omitempty"`
	GPUHourlyPrice       float64 `json:"gpu_hourly_price,omitempty"`
	Status               string  `json:"status,omitempty"`
}

// ListRegionsResponse is the API response for listing regions.
type ListRegionsResponse struct {
	Regions []Region `json:"regions"`
}

// ListRegions retrieves all regions matching the optional filters.
func (c *Client) ListRegions(ctx context.Context, country string, minRenewablePct float64, maxCarbonIntensity float64) ([]Region, error) {
	params := url.Values{}
	if country != "" {
		params.Set("country", country)
	}
	if minRenewablePct > 0 {
		params.Set("min_renewable_percentage", fmt.Sprintf("%.1f", minRenewablePct))
	}
	if maxCarbonIntensity > 0 {
		params.Set("max_carbon_intensity", fmt.Sprintf("%.1f", maxCarbonIntensity))
	}

	path := "/regions"
	if len(params) > 0 {
		path = fmt.Sprintf("/regions?%s", params.Encode())
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	result := &ListRegionsResponse{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("listing regions: %w", err)
	}

	return result.Regions, nil
}

// GetRegion retrieves a region by ID.
func (c *Client) GetRegion(ctx context.Context, id string) (*Region, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/regions/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &Region{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading region %s: %w", id, err)
	}

	return result, nil
}
