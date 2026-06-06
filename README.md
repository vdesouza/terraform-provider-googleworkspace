# Terraform Provider Google Workspace (Fork)

Community-maintained Terraform provider for Google Workspace resources.

This fork is under active development.

## Project Status

This repository is a fork of the original provider and is currently a work in progress.

Important expectations:
- The most actively maintained areas right now are Dynamic Groups and Chrome Policy resources.
- Other resources were inherited from the original project and have not yet been fully re-validated in this fork.
- Use with care in production and test changes in a non-production Google Workspace tenant first.

## Support Policy

- Best-effort community support.
- No formal SLA.
- Fixes are prioritized for recently changed areas (primarily Dynamic Groups and Chrome Policy resources).

## Recent Focus In This Fork

Recent development in this fork has focused on:
- Dynamic group behavior and creation/update reliability.
- Chrome Policy resources.
- Group and Org Unit policy lifecycle behavior.
- Additional target key handling.
- Retry and non-fatal error handling for API edge cases.
- Policy ordering and related Chrome policy workflow improvements.

See [CHANGELOG.md](CHANGELOG.md) for detailed release-by-release notes.

## Coverage and Validation

Current confidence level by area:
- Higher confidence: Dynamic groups and Chrome Policy related resources changed recently in this fork.
- Moderate confidence: Core inherited resources that have basic historical coverage but limited recent fork-specific re-testing.
- Lower confidence: Any inherited edge paths not touched recently by fork changes.

This does not mean other resources are broken, only that they are not all fully re-tested in this fork yet.

Recommended rollout approach:
- Start with read-only data sources or non-critical resources.
- Roll out Dynamic Groups and Chrome Policy changes first if those are your target areas.
- Apply with small plans and validate state transitions before scaling.

## Requirements

- Terraform >= 0.13
- Go >= 1.24 (for development/building this fork)
- Access to a Google Workspace environment for acceptance tests

## Build

```sh
make build
```

## Test

```sh
make test
```

Acceptance tests (require Google Workspace credentials/env vars):

```sh
make testacc
```

## Generate Documentation

```sh
make generate
```

Notes:
- Files under docs/ are generated output.
- Update source schemas/examples, then run make generate.

## Using This Provider

Before adopting broadly:
- Pin provider versions explicitly.
- Start with a small subset of resources.
- Validate import, update, and delete behavior in your environment.

## Contributing

Contributions and bug reports are welcome.

When opening issues, please include:
- Terraform version
- Provider version
- Resource/data source used
- Relevant config snippet (sanitized)
- Full error output

## Acknowledgements

- HashiCorp team for the original Terraform provider work.
- DeviaVir/terraform-provider-gsuite for early ecosystem inspiration.
