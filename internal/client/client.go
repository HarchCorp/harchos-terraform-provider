package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.harchos.io/v1"
	defaultTimeout = 30 * time.Second
)

// Client is an HTTP client for the HarchOS REST API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiKey     string
	region     string
	sovereignty string
	userAgent  string
}

// Config holds the configuration for constructing a new Client.
type Config struct {
	APIKey      string
	Region      string
	Sovereignty string
	BaseURL     string
}

// New creates a new HarchOS API client from the given configuration.
func New(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required for HarchOS client")
	}

	baseURLStr := cfg.BaseURL
	if baseURLStr == "" {
		baseURLStr = defaultBaseURL
	}

	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL %q: %w", baseURLStr, err)
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		apiKey:      cfg.APIKey,
		region:      cfg.Region,
		sovereignty: cfg.Sovereignty,
		userAgent:   "harchos-terraform-provider/0.1.0",
	}, nil
}

// APIError represents an error response from the HarchOS API.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HarchOS API error: status=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
}

// Workload represents a HarchOS workload resource.
type Workload struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Replicas    int               `json:"replicas"`
	Region      string            `json:"region"`
	Sovereignty string            `json:"sovereignty"`
	Env         map[string]string `json:"env,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Status      string            `json:"status,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
}

// Model represents a HarchOS model resource.
type Model struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Framework      string            `json:"framework"`
	Version        string            `json:"version"`
	SourceURI      string            `json:"source_uri"`
	Sovereignty    string            `json:"sovereignty"`
	Region         string            `json:"region"`
	Parameters     map[string]string `json:"parameters,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	Status         string            `json:"status,omitempty"`
	SizeBytes      int64             `json:"size_bytes,omitempty"`
	CreatedAt      string            `json:"created_at,omitempty"`
	UpdatedAt      string            `json:"updated_at,omitempty"`
}

// InferenceEndpoint represents a HarchOS inference endpoint.
type InferenceEndpoint struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	ModelID       string            `json:"model_id"`
	InstanceType  string            `json:"instance_type"`
	MinReplicas   int               `json:"min_replicas"`
	MaxReplicas   int               `json:"max_replicas"`
	Region        string            `json:"region"`
	Sovereignty   string            `json:"sovereignty"`
	EndpointURL   string            `json:"endpoint_url,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Status        string            `json:"status,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	UpdatedAt     string            `json:"updated_at,omitempty"`
}

// Dataset represents a HarchOS dataset resource.
type Dataset struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Format      string            `json:"format"`
	SizeBytes   int64             `json:"size_bytes"`
	Region      string            `json:"region"`
	Sovereignty string            `json:"sovereignty"`
	StoragePath string            `json:"storage_path,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Status      string            `json:"status,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
}

// NetworkPolicy represents a HarchOS network policy resource.
type NetworkPolicy struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Region      string             `json:"region"`
	Sovereignty string             `json:"sovereignty"`
	Ingress     []NetworkRule      `json:"ingress,omitempty"`
	Egress      []NetworkRule      `json:"egress,omitempty"`
	Tags        map[string]string  `json:"tags,omitempty"`
	Status      string             `json:"status,omitempty"`
	CreatedAt   string             `json:"created_at,omitempty"`
	UpdatedAt   string             `json:"updated_at,omitempty"`
}

// NetworkRule defines a single ingress or egress rule.
type NetworkRule struct {
	Protocol  string   `json:"protocol"`
	Port      int      `json:"port,omitempty"`
	From      []string `json:"from,omitempty"`
	To        []string `json:"to,omitempty"`
	Action    string   `json:"action"`
}

// StorageVolume represents a HarchOS storage volume resource.
type StorageVolume struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	SizeGB      int               `json:"size_gb"`
	VolumeType  string            `json:"volume_type"`
	Region      string            `json:"region"`
	Sovereignty string            `json:"sovereignty"`
	Encrypted   bool              `json:"encrypted"`
	Tags        map[string]string `json:"tags,omitempty"`
	Status      string            `json:"status,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
}

// Hub represents a HarchOS hub (compute cluster).
type Hub struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Region      string            `json:"region"`
	Sovereignty string            `json:"sovereignty"`
	Capacity    int               `json:"capacity"`
	Tags        map[string]string `json:"tags,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// ListHubsResponse is the API response for listing hubs.
type ListHubsResponse struct {
	Hubs []Hub `json:"hubs"`
}

// newRequest creates a new HTTP request with the appropriate headers.
func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing path %q: %w", path, err)
	}

	u := c.baseURL.ResolveReference(rel)

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		buf = bytes.NewBuffer(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-API-Key", c.apiKey)

	if c.region != "" {
		req.Header.Set("X-HarchOS-Region", c.region)
	}
	if c.sovereignty != "" {
		req.Header.Set("X-HarchOS-Sovereignty", c.sovereignty)
	}

	return req, nil
}

// do executes an HTTP request and decodes the response.
func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = fmt.Sprintf("request failed with status %d", resp.StatusCode)
		}
		return apiErr
	}

	if v != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// --- Workload CRUD ---

// CreateWorkload creates a new workload.
func (c *Client) CreateWorkload(ctx context.Context, workload *Workload) (*Workload, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/workloads", workload)
	if err != nil {
		return nil, err
	}

	result := &Workload{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("creating workload: %w", err)
	}

	return result, nil
}

// GetWorkload retrieves a workload by ID.
func (c *Client) GetWorkload(ctx context.Context, id string) (*Workload, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/workloads/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &Workload{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading workload %s: %w", id, err)
	}

	return result, nil
}

// UpdateWorkload updates an existing workload.
func (c *Client) UpdateWorkload(ctx context.Context, id string, workload *Workload) (*Workload, error) {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/workloads/%s", id), workload)
	if err != nil {
		return nil, err
	}

	result := &Workload{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("updating workload %s: %w", id, err)
	}

	return result, nil
}

// DeleteWorkload deletes a workload by ID.
func (c *Client) DeleteWorkload(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/workloads/%s", id), nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("deleting workload %s: %w", id, err)
	}

	return nil
}

// --- Model CRUD ---

// CreateModel creates a new model.
func (c *Client) CreateModel(ctx context.Context, model *Model) (*Model, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/models", model)
	if err != nil {
		return nil, err
	}

	result := &Model{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("creating model: %w", err)
	}

	return result, nil
}

// GetModel retrieves a model by ID.
func (c *Client) GetModel(ctx context.Context, id string) (*Model, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/models/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &Model{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading model %s: %w", id, err)
	}

	return result, nil
}

// UpdateModel updates an existing model.
func (c *Client) UpdateModel(ctx context.Context, id string, model *Model) (*Model, error) {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/models/%s", id), model)
	if err != nil {
		return nil, err
	}

	result := &Model{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("updating model %s: %w", id, err)
	}

	return result, nil
}

// DeleteModel deletes a model by ID.
func (c *Client) DeleteModel(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/models/%s", id), nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("deleting model %s: %w", id, err)
	}

	return nil
}

// --- InferenceEndpoint CRUD ---

// CreateInferenceEndpoint creates a new inference endpoint.
func (c *Client) CreateInferenceEndpoint(ctx context.Context, endpoint *InferenceEndpoint) (*InferenceEndpoint, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/inference-endpoints", endpoint)
	if err != nil {
		return nil, err
	}

	result := &InferenceEndpoint{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("creating inference endpoint: %w", err)
	}

	return result, nil
}

// GetInferenceEndpoint retrieves an inference endpoint by ID.
func (c *Client) GetInferenceEndpoint(ctx context.Context, id string) (*InferenceEndpoint, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/inference-endpoints/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &InferenceEndpoint{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading inference endpoint %s: %w", id, err)
	}

	return result, nil
}

// UpdateInferenceEndpoint updates an existing inference endpoint.
func (c *Client) UpdateInferenceEndpoint(ctx context.Context, id string, endpoint *InferenceEndpoint) (*InferenceEndpoint, error) {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/inference-endpoints/%s", id), endpoint)
	if err != nil {
		return nil, err
	}

	result := &InferenceEndpoint{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("updating inference endpoint %s: %w", id, err)
	}

	return result, nil
}

// DeleteInferenceEndpoint deletes an inference endpoint by ID.
func (c *Client) DeleteInferenceEndpoint(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/inference-endpoints/%s", id), nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("deleting inference endpoint %s: %w", id, err)
	}

	return nil
}

// --- Dataset CRUD ---

// CreateDataset creates a new dataset.
func (c *Client) CreateDataset(ctx context.Context, dataset *Dataset) (*Dataset, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/datasets", dataset)
	if err != nil {
		return nil, err
	}

	result := &Dataset{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("creating dataset: %w", err)
	}

	return result, nil
}

// GetDataset retrieves a dataset by ID.
func (c *Client) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/datasets/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &Dataset{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading dataset %s: %w", id, err)
	}

	return result, nil
}

// UpdateDataset updates an existing dataset.
func (c *Client) UpdateDataset(ctx context.Context, id string, dataset *Dataset) (*Dataset, error) {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/datasets/%s", id), dataset)
	if err != nil {
		return nil, err
	}

	result := &Dataset{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("updating dataset %s: %w", id, err)
	}

	return result, nil
}

// DeleteDataset deletes a dataset by ID.
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/datasets/%s", id), nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("deleting dataset %s: %w", id, err)
	}

	return nil
}

// --- NetworkPolicy CRUD ---

// CreateNetworkPolicy creates a new network policy.
func (c *Client) CreateNetworkPolicy(ctx context.Context, policy *NetworkPolicy) (*NetworkPolicy, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/network-policies", policy)
	if err != nil {
		return nil, err
	}

	result := &NetworkPolicy{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("creating network policy: %w", err)
	}

	return result, nil
}

// GetNetworkPolicy retrieves a network policy by ID.
func (c *Client) GetNetworkPolicy(ctx context.Context, id string) (*NetworkPolicy, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/network-policies/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &NetworkPolicy{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading network policy %s: %w", id, err)
	}

	return result, nil
}

// UpdateNetworkPolicy updates an existing network policy.
func (c *Client) UpdateNetworkPolicy(ctx context.Context, id string, policy *NetworkPolicy) (*NetworkPolicy, error) {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/network-policies/%s", id), policy)
	if err != nil {
		return nil, err
	}

	result := &NetworkPolicy{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("updating network policy %s: %w", id, err)
	}

	return result, nil
}

// DeleteNetworkPolicy deletes a network policy by ID.
func (c *Client) DeleteNetworkPolicy(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/network-policies/%s", id), nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("deleting network policy %s: %w", id, err)
	}

	return nil
}

// --- StorageVolume CRUD ---

// CreateStorageVolume creates a new storage volume.
func (c *Client) CreateStorageVolume(ctx context.Context, volume *StorageVolume) (*StorageVolume, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/storage-volumes", volume)
	if err != nil {
		return nil, err
	}

	result := &StorageVolume{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("creating storage volume: %w", err)
	}

	return result, nil
}

// GetStorageVolume retrieves a storage volume by ID.
func (c *Client) GetStorageVolume(ctx context.Context, id string) (*StorageVolume, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/storage-volumes/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &StorageVolume{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading storage volume %s: %w", id, err)
	}

	return result, nil
}

// UpdateStorageVolume updates an existing storage volume.
func (c *Client) UpdateStorageVolume(ctx context.Context, id string, volume *StorageVolume) (*StorageVolume, error) {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/storage-volumes/%s", id), volume)
	if err != nil {
		return nil, err
	}

	result := &StorageVolume{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("updating storage volume %s: %w", id, err)
	}

	return result, nil
}

// DeleteStorageVolume deletes a storage volume by ID.
func (c *Client) DeleteStorageVolume(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/storage-volumes/%s", id), nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("deleting storage volume %s: %w", id, err)
	}

	return nil
}

// --- Hub operations ---

// ListHubs retrieves all hubs matching the optional filter.
func (c *Client) ListHubs(ctx context.Context, region string) ([]Hub, error) {
	path := "/hubs"
	if region != "" {
		path = fmt.Sprintf("/hubs?region=%s", url.QueryEscape(region))
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	result := &ListHubsResponse{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("listing hubs: %w", err)
	}

	return result.Hubs, nil
}

// GetHub retrieves a hub by ID.
func (c *Client) GetHub(ctx context.Context, id string) (*Hub, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/hubs/%s", id), nil)
	if err != nil {
		return nil, err
	}

	result := &Hub{}
	if err := c.do(req, result); err != nil {
		return nil, fmt.Errorf("reading hub %s: %w", id, err)
	}

	return result, nil
}

// ParseID extracts a resource ID from a string, validating format.
func ParseID(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("id cannot be empty")
	}
	// HarchOS IDs follow the format: harchos_<type>_<alphanum>
	if len(s) < 10 {
		return "", fmt.Errorf("invalid id format: %s", s)
	}
	return s, nil
}

// ParseInt parses a string as an integer, returning a default if empty.
func ParseInt(s string, defaultVal int) (int, error) {
	if s == "" {
		return defaultVal, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", s, err)
	}
	return v, nil
}
