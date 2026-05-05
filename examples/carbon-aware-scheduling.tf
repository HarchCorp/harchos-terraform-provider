# HarchOS Carbon-Aware Scheduling Example
#
# This example demonstrates how to use the harchos_carbon_aware_schedule
# resource to enable carbon-aware workload scheduling with HarchOS.
#
# Carbon-aware scheduling automatically defers or relocates workloads
# based on real-time carbon intensity data, prioritizing green energy
# windows and low-carbon regions.

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
  sovereignty  = "strict"
}

variable "harchos_api_key" {
  description = "HarchOS API key"
  type        = string
  sensitive   = true
}

# ─── Workload ──────────────────────────────────────────────────────────────

resource "harchos_workload" "ml_training" {
  name        = "carbon-aware-ml-training"
  image       = "harchos/pytorch-training:v2.1"
  replicas    = 4
  region      = "morocco"
  sovereignty = "strict"

  env = {
    MODEL_NAME    = "llama-7b-finetune"
    EPOCHS        = "10"
    BATCH_SIZE    = "32"
    LEARNING_RATE = "0.0001"
  }

  tags = {
    project   = "carbon-aware-training"
    team      = "ml-platform"
    priority  = "high"
  }
}

# ─── Carbon-Aware Schedule ─────────────────────────────────────────────────

# Basic carbon-aware schedule: defer when carbon is high
resource "harchos_carbon_aware_schedule" "training_schedule" {
  workload_id           = harchos_workload.ml_training.id
  enabled               = true
  max_carbon_intensity  = 100    # gCO2/kWh - only run when intensity is below this
  preferred_region      = "morocco"
  defer_when_high_carbon = true  # Automatically defer when above threshold
  green_window_only     = false  # Run when below threshold, not just in green windows
  sovereignty           = "strict"
}

# ─── Strict Green-Window Schedule ──────────────────────────────────────────

# For workloads with the strictest carbon requirements:
# only run during identified green windows (periods of very low carbon intensity)
resource "harchos_workload" "batch_inference" {
  name        = "green-inference-batch"
  image       = "harchos/inference-batch:v1.5"
  replicas    = 2
  region      = "morocco"
  sovereignty = "regional"

  tags = {
    project = "green-inference"
    carbon  = "critical"
  }
}

resource "harchos_carbon_aware_schedule" "green_only_schedule" {
  workload_id           = harchos_workload.batch_inference.id
  enabled               = true
  max_carbon_intensity  = 50     # Very strict: only 50 gCO2/kWh
  preferred_region      = "morocco"
  defer_when_high_carbon = true
  green_window_only     = true   # Only run during green windows
  sovereignty           = "regional"
}

# ─── Flexible Cross-Region Schedule ────────────────────────────────────────

# For workloads that can run in any low-carbon region
resource "harchos_workload" "data_pipeline" {
  name        = "etl-pipeline"
  image       = "harchos/data-pipeline:v3.0"
  replicas    = 1
  sovereignty = "global"

  tags = {
    project = "data-platform"
  }
}

resource "harchos_carbon_aware_schedule" "flexible_schedule" {
  workload_id           = harchos_workload.data_pipeline.id
  enabled               = true
  max_carbon_intensity  = 200    # More relaxed threshold
  preferred_region      = "morocco"
  defer_when_high_carbon = true
  green_window_only     = false
  sovereignty           = "global"
}

# ─── Outputs ───────────────────────────────────────────────────────────────

output "training_schedule_id" {
  description = "ID of the ML training carbon-aware schedule"
  value       = harchos_carbon_aware_schedule.training_schedule.id
}

output "training_current_intensity" {
  description = "Current carbon intensity at the training workload's region"
  value       = harchos_carbon_aware_schedule.training_schedule.current_intensity
}

output "training_recommended_action" {
  description = "Recommended action from the carbon optimization engine"
  value       = harchos_carbon_aware_schedule.training_schedule.recommended_action
}

output "training_next_green_window" {
  description = "Start time of the next green window for the training workload"
  value       = harchos_carbon_aware_schedule.training_schedule.next_green_window
}

output "green_only_schedule_id" {
  description = "ID of the green-only inference schedule"
  value       = harchos_carbon_aware_schedule.green_only_schedule.id
}
