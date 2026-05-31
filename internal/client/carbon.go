package client

import (
        "context"
        "fmt"
        "net/http"
)

// CarbonAwareSchedule represents a carbon-aware scheduling configuration.
type CarbonAwareSchedule struct {
        ID                  string  `json:"id"`
        WorkloadID          string  `json:"workload_id"`
        Enabled             bool    `json:"enabled"`
        MaxCarbonIntensity  float64 `json:"max_carbon_intensity"`
        PreferredRegion     string  `json:"preferred_region"`
        DeferWhenHighCarbon bool    `json:"defer_when_high_carbon"`
        GreenWindowOnly     bool    `json:"green_window_only"`
        Sovereignty         string  `json:"sovereignty,omitempty"`
        Status              string  `json:"status,omitempty"`
        CurrentIntensity    float64 `json:"current_intensity,omitempty"`
        RecommendedAction   string  `json:"recommended_action,omitempty"`
        NextGreenWindow     string  `json:"next_green_window,omitempty"`
        CreatedAt           string  `json:"created_at,omitempty"`
        UpdatedAt           string  `json:"updated_at,omitempty"`
}

// CarbonForecast represents a carbon intensity forecast data point.
type CarbonForecast struct {
        Zone             string  `json:"zone"`
        Timestamp        string  `json:"timestamp"`
        CarbonIntensity  float64 `json:"carbon_intensity"`
        IsGreenWindow    bool    `json:"is_green_window"`
        OptimalHubID     string  `json:"optimal_hub_id,omitempty"`
        OptimalHubName   string  `json:"optimal_hub_name,omitempty"`
        DurationMinutes  int     `json:"duration_minutes,omitempty"`
}

// CarbonForecastResponse is the API response for carbon forecast.
type CarbonForecastResponse struct {
        Zone      string            `json:"zone"`
        Forecast  []CarbonForecast  `json:"forecast"`
        GreenWindows []GreenWindow  `json:"green_windows,omitempty"`
}

// GreenWindow represents a time window with low carbon intensity.
type GreenWindow struct {
        StartAt      string  `json:"start_at"`
        EndAt        string  `json:"end_at"`
        MinIntensity float64 `json:"min_intensity"`
        AvgIntensity float64 `json:"avg_intensity"`
        DurationMin  int     `json:"duration_min"`
}

// CarbonMetrics represents aggregate carbon metrics.
type CarbonMetrics struct {
        TotalCO2Grams       float64 `json:"total_co2_grams"`
        TotalEnergyKwh      float64 `json:"total_energy_kwh"`
        AvgIntensity        float64 `json:"avg_intensity"`
        ActiveSchedules     int     `json:"active_schedules"`
        DeferredWorkloads   int     `json:"deferred_workloads"`
        GreenWindowUsage    float64 `json:"green_window_usage_percent"`
        CarbonSavingsGrams  float64 `json:"carbon_savings_grams"`
        CarbonSavingsPercent float64 `json:"carbon_savings_percent"`
}

// CarbonOptimizeRequest is the request payload for carbon optimization.
type CarbonOptimizeRequest struct {
        WorkloadID          string  `json:"workload_id"`
        Enabled             bool    `json:"enabled"`
        MaxCarbonIntensity  float64 `json:"max_carbon_intensity"`
        PreferredRegion     string  `json:"preferred_region"`
        DeferWhenHighCarbon bool    `json:"defer_when_high_carbon"`
        GreenWindowOnly     bool    `json:"green_window_only"`
        Sovereignty         string  `json:"sovereignty,omitempty"`
}

// --- CarbonAwareSchedule CRUD ---

// CreateCarbonAwareSchedule creates a new carbon-aware schedule.
func (c *Client) CreateCarbonAwareSchedule(ctx context.Context, schedule *CarbonAwareSchedule) (*CarbonAwareSchedule, error) {
        // Server uses /carbon/optimize for carbon-aware scheduling
        optReq := &CarbonOptimizeRequest{
                WorkloadID:          schedule.WorkloadID,
                Enabled:             schedule.Enabled,
                MaxCarbonIntensity:  schedule.MaxCarbonIntensity,
                PreferredRegion:     schedule.PreferredRegion,
                DeferWhenHighCarbon: schedule.DeferWhenHighCarbon,
                GreenWindowOnly:     schedule.GreenWindowOnly,
                Sovereignty:         schedule.Sovereignty,
        }
        req, err := c.newRequest(ctx, http.MethodPost, "/carbon/optimize", optReq)
        if err != nil {
                return nil, err
        }

        result := &CarbonAwareSchedule{}
        if err := c.do(req, result); err != nil {
                return nil, fmt.Errorf("creating carbon-aware schedule: %w", err)
        }

        return result, nil
}

// GetCarbonAwareSchedule retrieves a carbon-aware schedule by ID.
func (c *Client) GetCarbonAwareSchedule(ctx context.Context, id string) (*CarbonAwareSchedule, error) {
        // NOTE: Server does not yet expose GET /carbon/schedules/{id}
        // This returns a placeholder for Terraform state management
        return &CarbonAwareSchedule{ID: id, Status: "active"}, nil
}

// UpdateCarbonAwareSchedule updates an existing carbon-aware schedule.
func (c *Client) UpdateCarbonAwareSchedule(ctx context.Context, id string, schedule *CarbonAwareSchedule) (*CarbonAwareSchedule, error) {
        // NOTE: Server does not yet expose PUT /carbon/schedules/{id}
        // Re-optimize with updated parameters
        return c.CreateCarbonAwareSchedule(ctx, schedule)
}

// DeleteCarbonAwareSchedule deletes a carbon-aware schedule by ID.
func (c *Client) DeleteCarbonAwareSchedule(ctx context.Context, id string) error {
        // NOTE: Server does not yet expose DELETE /carbon/schedules/{id}
        // Schedule is implicitly deleted when workload is removed
        return nil
}

// --- Carbon optimization and forecasting ---

// OptimizeWorkload sends an optimization request for a workload's carbon schedule.
func (c *Client) OptimizeWorkload(ctx context.Context, optReq *CarbonOptimizeRequest) (*CarbonAwareSchedule, error) {
        req, err := c.newRequest(ctx, http.MethodPost, "/carbon/optimize", optReq)
        if err != nil {
                return nil, err
        }

        result := &CarbonAwareSchedule{}
        if err := c.do(req, result); err != nil {
                return nil, fmt.Errorf("optimizing workload carbon schedule: %w", err)
        }

        return result, nil
}

// GetCarbonForecast retrieves the carbon intensity forecast for a zone.
func (c *Client) GetCarbonForecast(ctx context.Context, zone string, hours int) (*CarbonForecastResponse, error) {
        path := fmt.Sprintf("/carbon/forecast/%s?hours=%d", zone, hours)
        req, err := c.newRequest(ctx, http.MethodGet, path, nil)
        if err != nil {
                return nil, err
        }

        result := &CarbonForecastResponse{}
        if err := c.do(req, result); err != nil {
                return nil, fmt.Errorf("reading carbon forecast for zone %s: %w", zone, err)
        }

        return result, nil
}

// GetCarbonMetrics retrieves aggregate carbon metrics for the platform.
func (c *Client) GetCarbonMetrics(ctx context.Context) (*CarbonMetrics, error) {
        req, err := c.newRequest(ctx, http.MethodGet, "/carbon/metrics", nil)
        if err != nil {
                return nil, err
        }

        result := &CarbonMetrics{}
        if err := c.do(req, result); err != nil {
                return nil, fmt.Errorf("reading carbon metrics: %w", err)
        }

        return result, nil
}
