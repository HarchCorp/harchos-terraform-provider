# Changelog

All notable changes to the harchos Terraform provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-12-15

### Added

- Initial release of the HarchOS Terraform provider
- Provider configuration with `api_key`, `region`, `sovereignty`, and `base_url` attributes
- Environment variable fallbacks: `HARCHOS_API_KEY`, `HARCHOS_REGION`, `HARCHOS_SOVEREIGNTY`, `HARCHOS_BASE_URL`
- **Resources:**
  - `harchos_workload` — Containerized compute workloads with replicas and env vars
  - `harchos_model` — ML model registry with framework validation
  - `harchos_inference_endpoint` — Real-time model serving with auto-scaling
  - `harchos_dataset` — Sovereign data stores with format validation
  - `harchos_network_policy` — Zero-trust networking with ingress/egress rules
  - `harchos_storage_volume` — Persistent block storage with encryption
- **Data Sources:**
  - `harchos_hubs` — List available compute hubs with region filter
  - `harchos_workload` — Read existing workload by ID
  - `harchos_model` — Read existing model by ID
- **Sovereignty Enforcement:**
  - Three levels: `strict`, `regional`, `global`
  - Downgrade prevention enforced at plan time
  - Provider-level sovereignty acts as a floor
  - Effective sovereignty always picks the more restrictive level
- **Drift Detection:**
  - Plan-time drift detection on all resources via `ModifyPlan`
  - 404-tolerant reads with automatic state removal
- **Import Support:**
  - All resources support `terraform import` via ID passthrough
- CI/CD pipeline with build, test, lint, and acceptance test jobs
- Apache 2.0 license

[0.1.0]: https://github.com/HarchCorp/harchos-terraform-provider/releases/tag/v0.1.0
