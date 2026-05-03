# Contributing to HarchOS Terraform Provider

First off, thank you for considering contributing to the HarchOS Terraform provider! It's people like you that make the HarchOS ecosystem thrive.

## Code of Conduct

This project and everyone participating in it is governed by the [HarchCorp Code of Conduct](https://github.com/HarchCorp/.github/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Terraform version** (`terraform version`)
- **Provider version**
- **Affected resource(s) and/or data source(s)**
- **Terraform configuration** that reproduces the issue
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Debug output** (`TF_LOG=DEBUG terraform apply`)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. Include:

- **Use case** — why is this needed?
- **Expected behavior** — what should happen?
- **Current behavior** — what happens instead?
- **Proposed API** — example Terraform configuration showing how the feature would be used

### Pull Requests

1. **Fork** the repository
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Make your changes** following the coding standards below
4. **Add tests** — unit tests for all new logic, acceptance tests for new resources
5. **Run the test suite**:
   ```bash
   make check
   ```
6. **Commit** with a descriptive message following [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat: add harchos_queue resource
   fix: prevent sovereignty downgrade on model update
   docs: add example for inference endpoint
   ```
7. **Push** and open a Pull Request against `main`

## Development Setup

### Prerequisites

- Go 1.21+
- Terraform 1.5+
- Make (optional)

### Building

```bash
make build
```

### Testing

```bash
# Unit tests
make test

# Acceptance tests (requires API credentials)
export HARCHOS_API_KEY="your-key"
export HARCHOS_REGION="eu-west-1"
make testacc
```

### Local Development Install

```bash
make dev
```

This installs the provider binary to your local Terraform plugin directory for testing.

## Coding Standards

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Run `gofmt` before committing
- All exported functions must have documentation comments
- Use `context.Context` as the first parameter in all API calls
- Handle errors explicitly — do not use `_` to discard errors

### Terraform Provider Specific

- All resources **must** implement `ResourceWithModifyPlan` for drift detection
- All resources **must** support import via `ResourceWithImportState`
- All resources **must** enforce sovereignty escalation policy
- Use `terraform-plugin-framework` (not SDKv2) for all new resources
- Schema descriptions must be written in Markdown
- Computed attributes should use `UseStateForUnknown` plan modifiers

### Sovereignty

Sovereignty enforcement is a **critical compliance requirement**. Any changes to the sovereignty validation logic require explicit review by the CODEOWNERS. Never:

- Remove sovereignty downgrade prevention
- Allow a less restrictive effective sovereignty than the resource's current level
- Skip sovereignty checks in the `ModifyPlan` method

### Testing

- Unit tests for all validation logic (sovereignty, effective level calculation)
- Acceptance tests for full CRUD lifecycle of each resource
- Acceptance test for sovereignty downgrade prevention
- Import state verification for all resources

## Release Process

Releases are managed by maintainers using [goreleaser](https://goreleaser.com/):

1. Update `CHANGELOG.md` with the new version
2. Tag the release: `git tag v0.x.0`
3. Push the tag: `git push origin v0.x.0`
4. Goreleaser builds and publishes the release

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
