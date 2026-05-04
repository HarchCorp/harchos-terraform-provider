package client

import (
        "context"
        "encoding/json"
        "errors"
        "fmt"
        "net/http"
        "net/http/httptest"
        "strconv"
        "testing"
)

// --- ParseID tests ---

func TestParseID_Valid(t *testing.T) {
        id, err := ParseID("harchos_workload_abc123def")
        if err != nil {
                t.Fatalf("expected no error, got %v", err)
        }
        if id != "harchos_workload_abc123def" {
                t.Fatalf("expected id to be returned, got %q", id)
        }
}

func TestParseID_TooShort(t *testing.T) {
        _, err := ParseID("short")
        if err == nil {
                t.Fatal("expected error for short ID, got nil")
        }
}

func TestParseID_Empty(t *testing.T) {
        _, err := ParseID("")
        if err == nil {
                t.Fatal("expected error for empty ID, got nil")
        }
}

func TestParseID_ExactlyTen(t *testing.T) {
        id, err := ParseID("harchos_12")
        if err != nil {
                t.Fatalf("expected no error for 10-char ID, got %v", err)
        }
        if id != "harchos_12" {
                t.Fatalf("unexpected id %q", id)
        }
}

func TestParseID_NineChars(t *testing.T) {
        _, err := ParseID("harchos_1")
        if err == nil {
                t.Fatal("expected error for 9-char ID, got nil")
        }
}

// --- ParseInt tests ---

func TestParseInt_Valid(t *testing.T) {
        v, err := ParseInt("42", 0)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if v != 42 {
                t.Fatalf("expected 42, got %d", v)
        }
}

func TestParseInt_Zero(t *testing.T) {
        v, err := ParseInt("0", 99)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if v != 0 {
                t.Fatalf("expected 0, got %d", v)
        }
}

func TestParseInt_Negative(t *testing.T) {
        v, err := ParseInt("-5", 0)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if v != -5 {
                t.Fatalf("expected -5, got %d", v)
        }
}

func TestParseInt_EmptyReturnsDefault(t *testing.T) {
        v, err := ParseInt("", 7)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if v != 7 {
                t.Fatalf("expected default 7, got %d", v)
        }
}

func TestParseInt_Invalid(t *testing.T) {
        _, err := ParseInt("notanumber", 0)
        if err == nil {
                t.Fatal("expected error for invalid integer, got nil")
        }
}

// --- APIError.Error() tests ---

func TestAPIError_Error(t *testing.T) {
        e := &APIError{
                StatusCode: 400,
                Code:       "bad_request",
                Message:    "invalid parameter",
        }
        msg := e.Error()
        if msg == "" {
                t.Fatal("Error() should not return empty string")
        }
}

func TestAPIError_ErrorWithDetails(t *testing.T) {
        e := &APIError{
                StatusCode: 400,
                Code:       "bad_request",
                Message:    "invalid parameter",
                Details:    "field 'name' is required",
        }
        msg := e.Error()
        if msg == "" {
                t.Fatal("Error() should not return empty string")
        }
        // Details field exists but Error() format doesn't include it; just verify no panic
}

// --- Client option / construction tests ---

func TestNew_MissingAPIKey(t *testing.T) {
        _, err := New(Config{})
        if err == nil {
                t.Fatal("expected error when APIKey is missing")
        }
}

func TestNew_ValidConfig(t *testing.T) {
        c, err := New(Config{
                APIKey:  "test-key",
                Region:  "eu-west-1",
                BaseURL: "https://api.example.com/v1",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if c.apiKey != "test-key" {
                t.Fatalf("expected apiKey test-key, got %q", c.apiKey)
        }
        if c.region != "eu-west-1" {
                t.Fatalf("expected region eu-west-1, got %q", c.region)
        }
}

func TestNew_DefaultBaseURL(t *testing.T) {
        c, err := New(Config{APIKey: "test-key"})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if c.baseURL.String() != defaultBaseURL {
                t.Fatalf("expected default base URL %q, got %q", defaultBaseURL, c.baseURL.String())
        }
}

func TestNew_InvalidBaseURL(t *testing.T) {
        _, err := New(Config{
                APIKey:  "test-key",
                BaseURL: "://invalid",
        })
        if err == nil {
                t.Fatal("expected error for invalid base URL")
        }
}

func TestNew_Sovereignty(t *testing.T) {
        c, err := New(Config{
                APIKey:      "test-key",
                Sovereignty: "strict",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if c.sovereignty != "strict" {
                t.Fatalf("expected sovereignty strict, got %q", c.sovereignty)
        }
}

// --- Request construction tests ---

func TestNewRequest_Headers(t *testing.T) {
        c, err := New(Config{
                APIKey:  "my-api-key",
                Region:  "us-east-1",
                BaseURL: "https://api.example.com/v1",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        req, err := c.newRequest(context.Background(), http.MethodGet, "/test", nil)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        if req.Header.Get("Content-Type") != "application/json" {
                t.Fatalf("expected Content-Type application/json, got %q", req.Header.Get("Content-Type"))
        }
        if req.Header.Get("Accept") != "application/json" {
                t.Fatalf("expected Accept application/json, got %q", req.Header.Get("Accept"))
        }
        if req.Header.Get("X-API-Key") != "my-api-key" {
                t.Fatalf("expected X-API-Key my-api-key, got %q", req.Header.Get("X-API-Key"))
        }
        if req.Header.Get("X-HarchOS-Region") != "us-east-1" {
                t.Fatalf("expected X-HarchOS-Region us-east-1, got %q", req.Header.Get("X-HarchOS-Region"))
        }
}

func TestNewRequest_URLBuilding(t *testing.T) {
        c, err := New(Config{
                APIKey:  "key",
                BaseURL: "https://api.example.com/v1",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        req, err := c.newRequest(context.Background(), http.MethodGet, "/workloads", nil)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        expected := "https://api.example.com/workloads"
        if req.URL.String() != expected {
                t.Fatalf("expected URL %q, got %q", expected, req.URL.String())
        }
}

func TestNewRequest_AuthInjection(t *testing.T) {
        c, err := New(Config{
                APIKey:      "secret-key",
                Region:      "eu-west-1",
                Sovereignty: "strict",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        req, err := c.newRequest(context.Background(), http.MethodGet, "/test", nil)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        if req.Header.Get("X-API-Key") != "secret-key" {
                t.Fatalf("expected X-API-Key secret-key, got %q", req.Header.Get("X-API-Key"))
        }
        if req.Header.Get("X-HarchOS-Sovereignty") != "strict" {
                t.Fatalf("expected X-HarchOS-Sovereignty strict, got %q", req.Header.Get("X-HarchOS-Sovereignty"))
        }
}

func TestNewRequest_NoRegionHeaderWhenEmpty(t *testing.T) {
        c, err := New(Config{
                APIKey: "key",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        req, err := c.newRequest(context.Background(), http.MethodGet, "/test", nil)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        if req.Header.Get("X-HarchOS-Region") != "" {
                t.Fatalf("expected no X-HarchOS-Region header, got %q", req.Header.Get("X-HarchOS-Region"))
        }
}

// --- HTTP error mapping tests ---

func TestHTTPErrorMapping(t *testing.T) {
        tests := []struct {
                statusCode int
                code       string
                message    string
        }{
                {400, "bad_request", "invalid input"},
                {401, "unauthorized", "invalid API key"},
                {403, "forbidden", "access denied"},
                {404, "not_found", "resource not found"},
                {409, "conflict", "resource already exists"},
                {429, "rate_limited", "too many requests"},
                {500, "internal_error", "server error"},
                {503, "service_unavailable", "service unavailable"},
        }

        for _, tc := range tests {
                t.Run(strconv.Itoa(tc.statusCode), func(t *testing.T) {
                        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                                w.Header().Set("Content-Type", "application/json")
                                w.WriteHeader(tc.statusCode)
                                _ = json.NewEncoder(w).Encode(map[string]string{
                                        "code":    tc.code,
                                        "message": tc.message,
                                })
                        }))
                        defer server.Close()

                        c, err := New(Config{
                                APIKey:  "test-key",
                                BaseURL: server.URL,
                        })
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }

                        _, err = c.GetWorkload(context.Background(), "test-id")
                        if err == nil {
                                t.Fatalf("expected error for status %d", tc.statusCode)
                        }

                        var apiErr *APIError
                        if !errors.As(err, &apiErr) {
                                t.Fatalf("expected *APIError, got %T: %v", err, err)
                        }
                        if apiErr.StatusCode != tc.statusCode {
                                t.Fatalf("expected status %d, got %d", tc.statusCode, apiErr.StatusCode)
                        }
                })
        }
}

// --- CRUD tests against mock server ---

func TestCreateWorkload(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        t.Fatalf("expected POST, got %s", r.Method)
                }
                if r.URL.Path != "/workloads" {
                        t.Fatalf("expected path /workloads, got %s", r.URL.Path)
                }

                var wl Workload
                if err := json.NewDecoder(r.Body).Decode(&wl); err != nil {
                        t.Fatalf("error decoding body: %v", err)
                }

                wl.ID = "harchos_workload_new"
                wl.Status = "running"
                wl.CreatedAt = "2024-01-01T00:00:00Z"
                wl.UpdatedAt = "2024-01-01T00:00:00Z"

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(wl)
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        result, err := c.CreateWorkload(context.Background(), &Workload{
                Name:     "test-workload",
                Image:    "nginx:latest",
                Replicas: 2,
                Region:   "eu-west-1",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result.ID != "harchos_workload_new" {
                t.Fatalf("expected ID harchos_workload_new, got %q", result.ID)
        }
        if result.Status != "running" {
                t.Fatalf("expected status running, got %q", result.Status)
        }
}

func TestGetWorkload(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        t.Fatalf("expected GET, got %s", r.Method)
                }
                expectedPath := "/workloads/test-id"
                if r.URL.Path != expectedPath {
                        t.Fatalf("expected path %s, got %s", expectedPath, r.URL.Path)
                }

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(Workload{
                        ID:     "test-id",
                        Name:   "my-workload",
                        Status: "running",
                })
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        result, err := c.GetWorkload(context.Background(), "test-id")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result.Name != "my-workload" {
                t.Fatalf("expected name my-workload, got %q", result.Name)
        }
}

func TestUpdateWorkload(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPut {
                        t.Fatalf("expected PUT, got %s", r.Method)
                }

                var wl Workload
                _ = json.NewDecoder(r.Body).Decode(&wl)
                wl.Status = "running"
                wl.UpdatedAt = "2024-01-02T00:00:00Z"

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(wl)
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        result, err := c.UpdateWorkload(context.Background(), "test-id", &Workload{
                Name:     "updated-workload",
                Replicas: 5,
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result.Name != "updated-workload" {
                t.Fatalf("expected name updated-workload, got %q", result.Name)
        }
}

func TestDeleteWorkload(t *testing.T) {
        called := false
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodDelete {
                        t.Fatalf("expected DELETE, got %s", r.Method)
                }
                called = true
                w.WriteHeader(http.StatusNoContent)
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        err = c.DeleteWorkload(context.Background(), "test-id")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if !called {
                t.Fatal("expected DELETE to be called")
        }
}

func TestListHubs(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        t.Fatalf("expected GET, got %s", r.Method)
                }

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(ListHubsResponse{
                        Hubs: []Hub{
                                {ID: "hub-1", Name: "eu-hub", Region: "eu-west-1", Capacity: 100},
                                {ID: "hub-2", Name: "us-hub", Region: "us-east-1", Capacity: 200},
                        },
                })
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        hubs, err := c.ListHubs(context.Background(), "")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if len(hubs) != 2 {
                t.Fatalf("expected 2 hubs, got %d", len(hubs))
        }
}

func TestListHubs_WithRegion(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                region := r.URL.Query().Get("region")
                if region != "eu-west-1" {
                        t.Fatalf("expected region query param eu-west-1, got %q", region)
                }

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(ListHubsResponse{
                        Hubs: []Hub{
                                {ID: "hub-1", Name: "eu-hub", Region: "eu-west-1"},
                        },
                })
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        hubs, err := c.ListHubs(context.Background(), "eu-west-1")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if len(hubs) != 1 {
                t.Fatalf("expected 1 hub, got %d", len(hubs))
        }
}

func TestGetNetworkPolicy(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(NetworkPolicy{
                        ID:          "np-1",
                        Name:        "test-policy",
                        Region:      "eu-west-1",
                        Sovereignty: "strict",
                        Ingress: []NetworkRule{
                                {Protocol: "tcp", Port: 443, Action: "allow"},
                        },
                        Egress: []NetworkRule{
                                {Protocol: "tcp", Port: 80, Action: "deny"},
                        },
                        Status: "active",
                })
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        result, err := c.GetNetworkPolicy(context.Background(), "np-1")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if len(result.Ingress) != 1 {
                t.Fatalf("expected 1 ingress rule, got %d", len(result.Ingress))
        }
        if len(result.Egress) != 1 {
                t.Fatalf("expected 1 egress rule, got %d", len(result.Egress))
        }
        if result.Ingress[0].Port != 443 {
                t.Fatalf("expected ingress port 443, got %d", result.Ingress[0].Port)
        }
}

func TestCreateModel(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                var m Model
                _ = json.NewDecoder(r.Body).Decode(&m)
                m.ID = "model-123"
                m.Status = "ready"

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(m)
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        result, err := c.CreateModel(context.Background(), &Model{
                Name:      "test-model",
                Framework: "pytorch",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result.ID != "model-123" {
                t.Fatalf("expected ID model-123, got %q", result.ID)
        }
}

func TestCreateStorageVolume(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                var sv StorageVolume
                _ = json.NewDecoder(r.Body).Decode(&sv)
                sv.ID = "vol-123"
                sv.Status = "available"

                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(sv)
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        result, err := c.CreateStorageVolume(context.Background(), &StorageVolume{
                Name:       "test-vol",
                SizeGB:     100,
                VolumeType: "ssd",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result.ID != "vol-123" {
                t.Fatalf("expected ID vol-123, got %q", result.ID)
        }
}

func TestAPINoBodyOnNoContent(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusNoContent)
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        err = c.DeleteWorkload(context.Background(), "test-id")
        if err != nil {
                t.Fatalf("unexpected error on NoContent: %v", err)
        }
}

func TestAPIErrorUnparseableBody(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusInternalServerError)
                fmt.Fprint(w, "not json")
        }))
        defer server.Close()

        c, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        _, err = c.GetWorkload(context.Background(), "test-id")
        if err == nil {
                t.Fatal("expected error, got nil")
        }

        var apiErr *APIError
        if !errors.As(err, &apiErr) {
                t.Fatalf("expected *APIError, got %T: %v", err, err)
        }
        if apiErr.StatusCode != 500 {
                t.Fatalf("expected status 500, got %d", apiErr.StatusCode)
        }
}
