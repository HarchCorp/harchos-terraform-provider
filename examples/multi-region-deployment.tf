# HarchOS Multi-Region Deployment Example
#
# This example demonstrates how to deploy workloads across multiple
# HarchOS regions with carbon-aware scheduling, leveraging the
# harchos_regions data source to dynamically select optimal regions
# based on carbon intensity and renewable energy availability.

terraform {
  required_providers {
    harchos = {
      source  = "registry.terraform.io/HarchCorp/harchos"
      version = "~> 1.0"
    }
  }
}

provider "harchos" {
  api_key      = var.harchos_api_key
  region       = "morocco"
  sovereignty  = "regional"
}

variable "harchos_api_key" {
  description = "HarchOS API key"
  type        = string
  sensitive   = true
}

# ─── Discover Low-Carbon Regions ───────────────────────────────────────────

# Find all regions with high renewable energy percentage
data "harchos_regions" "green_regions" {
  min_renewable_percentage = 60
  max_carbon_intensity     = 150
}

# Find Moroccan regions specifically
data "harchos_regions" "morocco_regions" {
  country = "Morocco"
}

# ─── Primary Region Workload ───────────────────────────────────────────────

resource "harchos_workload" "primary_api" {
  name        = "harchos-api-primary"
  image       = "harchos/api-server:v2.4"
  replicas    = 3
  region      = "morocco"
  sovereignty = "regional"

  env = {
    NODE_ENV      = "production"
    LOG_LEVEL     = "info"
    METRICS_PORT  = "9090"
  }

  tags = {
    service   = "api"
    tier      = "primary"
    region    = "morocco"
  }
}

resource "harchos_carbon_aware_schedule" "primary_schedule" {
  workload_id            = harchos_workload.primary_api.id
  enabled                = true
  max_carbon_intensity   = 100
  preferred_region       = "morocco"
  defer_when_high_carbon = true
  green_window_only      = false
  sovereignty            = "regional"
}

# ─── Secondary Region Workload ─────────────────────────────────────────────

# Deploy a secondary instance in a low-carbon region
# The region is dynamically chosen from green_regions data source

resource "harchos_workload" "secondary_api" {
  name        = "harchos-api-secondary"
  image       = "harchos/api-server:v2.4"
  replicas    = 2
  sovereignty = "regional"

  env = {
    NODE_ENV      = "production"
    LOG_LEVEL     = "info"
    METRICS_PORT  = "9090"
    PRIMARY       = "false"
  }

  tags = {
    service = "api"
    tier    = "secondary"
  }
}

resource "harchos_carbon_aware_schedule" "secondary_schedule" {
  workload_id            = harchos_workload.secondary_api.id
  enabled                = true
  max_carbon_intensity   = 150
  preferred_region       = "morocco"
  defer_when_high_carbon = true
  green_window_only      = false
  sovereignty            = "regional"
}

# ─── Inference Endpoint (Multi-Region) ─────────────────────────────────────

resource "harchos_model" "llama_inference" {
  name        = "llama-7b-chat"
  framework   = "pytorch"
  version     = "v1.0.0"
  source_uri  = "harchos://models/llama-7b-chat/v1.0.0"
  sovereignty = "regional"

  tags = {
    type     = "llm"
    use_case = "chat"
  }
}

resource "harchos_inference_endpoint" "primary_inference" {
  name          = "llama-chat-primary"
  model_id      = harchos_model.llama_inference.id
  instance_type = "gpu.large"
  min_replicas  = 2
  max_replicas  = 8
  region        = "morocco"
  sovereignty   = "regional"

  tags = {
    service = "inference"
    tier    = "primary"
  }
}

# ─── Storage Volumes ───────────────────────────────────────────────────────

resource "harchos_storage_volume" "model_cache" {
  name        = "model-cache-volume"
  size_gb     = 500
  volume_type = "ssd"
  region      = "morocco"
  sovereignty = "regional"
  encrypted   = true

  tags = {
    purpose = "model-cache"
  }
}

resource "harchos_storage_volume" "training_data" {
  name        = "training-data-volume"
  size_gb     = 2000
  volume_type = "nvme"
  region      = "morocco"
  sovereignty = "strict"
  encrypted   = true

  tags = {
    purpose = "training-data"
  }
}

# ─── Network Policies ──────────────────────────────────────────────────────

resource "harchos_network_policy" "api_policy" {
  name        = "api-network-policy"
  region      = "morocco"
  sovereignty = "regional"

  ingress {
    protocol = "tcp"
    port     = 443
    action   = "allow"
  }

  egress {
    protocol = "tcp"
    port     = 443
    action   = "allow"
  }

  tags = {
    service = "api"
  }
}

# ─── Outputs ───────────────────────────────────────────────────────────────

output "green_regions" {
  description = "List of regions with >60% renewable energy and <150 gCO2/kWh"
  value       = data.harchos_regions.green_regions.regions
}

output "morocco_regions" {
  description = "List of HarchOS regions in Morocco"
  value       = data.harchos_regions.morocco_regions.regions
}

output "primary_workload_id" {
  description = "ID of the primary API workload"
  value       = harchos_workload.primary_api.id
}

output "primary_carbon_status" {
  description = "Carbon-aware status of the primary workload"
  value = {
    current_intensity  = harchos_carbon_aware_schedule.primary_schedule.current_intensity
    recommended_action = harchos_carbon_aware_schedule.primary_schedule.recommended_action
    next_green_window  = harchos_carbon_aware_schedule.primary_schedule.next_green_window
  }
}

output "inference_endpoint_url" {
  description = "URL of the primary inference endpoint"
  value       = harchos_inference_endpoint.primary_inference.endpoint_url
}

output "model_cache_volume_id" {
  description = "ID of the model cache storage volume"
  value       = harchos_storage_volume.model_cache.id
}
