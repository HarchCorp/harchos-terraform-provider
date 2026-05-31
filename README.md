# HarchOS Terraform Provider

[![CI](https://github.com/HarchCorp/harchos-terraform-provider/actions/workflows/ci.yml/badge.svg)](https://github.com/HarchCorp/harchos-terraform-provider/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

The **harchos** Terraform provider is the official provider for [HarchCorp](https://harchcorp.com)'s HarchOS infrastructure platform. It enables infrastructure-as-code management of HarchOS compute workloads, ML models, inference endpoints, datasets, network policies, and storage volumes with built-in data sovereignty enforcement.

## Quick Start

### Installation

```terraform
terraform {
  required_providers {
    harchos = {
      source  = "HarchCorp/harchos"
      version = "~> 0.1"
    }
  }
}

provider "harchos" {
  api_key      = var.harchos_api_key
  region       = "eu-west-1"
  sovereignty  = "regional"
}
```

### Authentication

Configure the provider using either attributes or environment variables:

| Attribute   | Environment Variable   | Required |
|-------------|------------------------|----------|
| `api_key`   | `HARCHOS_API_KEY`      | Yes      |
| `region`    | `HARCHOS_REGION`       | Yes      |
| `sovereignty` | `HARCHOS_SOVEREIGNTY` | No       |
| `base_url`  | `HARCHOS_BASE_URL`     | No       |

## Resources

| Resource | Description |
|----------|-------------|
| `harchos_workload` | Containerized compute workloads |
| `harchos_model` | ML/AI model registry entries |
| `harchos_inference_endpoint` | Real-time model serving with auto-scaling |
| `harchos_dataset` | Sovereign data stores |
| `harchos_network_policy` | Zero-trust network rules |
| `harchos_storage_volume` | Persistent block storage |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `harchos_hubs` | List available compute hubs |
| `harchos_workload` | Read an existing workload |
| `harchos_model` | Read an existing model |

## Sovereignty Enforcement

HarchOS enforces a strict data sovereignty model with three levels:

| Level | Description |
|-------|-------------|
| **strict** | Data and compute cannot leave the designated region. No cross-region replication. |
| **regional** | Data and compute may replicate within the same geographic region. |
| **global** | Data and compute may replicate across regions globally. |

### Key Rules

1. **Sovereignty cannot be downgraded** — A resource created at `strict` cannot be changed to `regional` or `global`. This is enforced at Terraform plan time.
2. **Provider-level sovereignty is a floor** — If the provider has `sovereignty = "regional"`, resources can be `regional` or `strict`, but not `global`.
3. **Effective sovereignty** — The more restrictive of provider-level and resource-level sovereignty always applies.

### Example

```terraform
provider "harchos" {
  api_key     = var.api_key
  region      = "eu-west-1"
  sovereignty = "regional"  # Floor: resources must be regional or strict
}

# This works — strict is more restrictive than the provider's regional
resource "harchos_workload" "sensitive" {
  name        = "gdpr-workload"
  image       = "myapp:v1"
  sovereignty = "strict"
}

# This would FAIL — cannot downgrade from strict to regional
# resource "harchos_workload" "sensitive" {
#   sovereignty = "regional"  # Error: sovereignty cannot be downgraded
# }
```

## Example Configuration

```terraform
# Deploy a model and create an inference endpoint
resource "harchos_model" "sentiment" {
  name        = "sentiment-analysis"
  framework   = "pytorch"
  version     = "v2.1.0"
  source_uri  = "harchos://models/sentiment/v2.1.0"
  sovereignty = "strict"

  parameters = {
    max_seq_length = "512"
    batch_size     = "32"
  }
}

resource "harchos_inference_endpoint" "sentiment_api" {
  name          = "sentiment-api"
  model_id      = harchos_model.sentiment.id
  instance_type = "gpu.medium"
  min_replicas  = 2
  max_replicas  = 10
  sovereignty   = "strict"
}

# Storage for training data
resource "harchos_storage_volume" "training_data" {
  name        = "training-data-vol"
  size_gb     = 500
  volume_type = "ssd"
  encrypted   = true
  sovereignty = "strict"
}
```

## Development

### Requirements

- [Go](https://golang.org/) 1.21+
- [Terraform](https://www.terraform.io/downloads.html) 1.5+
- Make (optional)

### Building

```bash
make build
```

### Testing

Unit tests:
```bash
make test
```

Acceptance tests (requires `HARCHOS_API_KEY`):
```bash
export HARCHOS_API_KEY="your-api-key"
export HARCHOS_REGION="eu-west-1"
make testacc
```

### Generating Documentation

```bash
make docs
```

## License

This project is licensed under the Apache License 2.0 — see the [LICENSE](LICENSE) file for details.
