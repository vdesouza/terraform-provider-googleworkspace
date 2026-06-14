# Terraform Provider Google Workspace (Fork)

Community-maintained Terraform provider for Google Workspace resources.

This fork is under active development.

## Project Status

This repository is a fork of the original provider and is currently a work in progress.

Important expectations:
- The most actively maintained areas right now are Dynamic Groups and Chrome Policy resources.
- Other resources were inherited from the original project and have not yet been fully re-validated in this fork.
- Use with care in production and test changes in a non-production Google Workspace tenant first.

## Recent Focus In This Fork

Recent development in this fork has focused on:
- Dynamic group behavior and creation/update reliability.
- Chrome Policy resources.
- Group and Org Unit policy lifecycle behavior.
- Additional target key handling.
- Retry and non-fatal error handling for API edge cases.
- Policy ordering and related Chrome policy workflow improvements.

See [CHANGELOG.md](CHANGELOG.md) for detailed release-by-release notes.

## Requirements

- Terraform >= 1.4
- Go >= 1.24 (for development/building this fork)
- Access to a Google Workspace environment

## Build

```sh
make build
```

## Test

```sh
make test
```

Tests (require Google Workspace credentials/env vars):

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

## Companion Modules

This repository also ships a set of YAML-driven Terraform modules that compose provider resources into higher-level workflows for Chrome management. They live under [`modules/`](modules/) and can be consumed via a Git source pinned to a release tag:

```hcl
module "chrome_policies" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/policies?ref=v1.4.0"
  # ...
}
```

| Module | Purpose |
| --- | --- |
| [`variables`](modules/variables/) | Centralized YAML variable substitution for the other modules. |
| [`groups`](modules/groups/) | Static and dynamic Google Workspace groups from YAML. |
| [`assets`](modules/assets/) | File uploads to Chrome Policy storage (wallpapers, avatars, ToS). |
| [`policies`](modules/policies/) | Chrome policies for groups and OUs, with asset reference resolution. |
| [`extensions`](modules/extensions/) | A variation on `policies` for managing Chrome extensions, Android apps, and web apps deployed to groups or OUs. |
| [`group_priority`](modules/group_priority/) | Resolves ordering when multiple groups assign overlapping policies/extensions. |

See [`modules/README.md`](modules/README.md) for the dependency graph and reference configurations.

## Contributing

Contributions and bug reports are welcome.

When opening issues, please include:

- Terraform version
- Provider version
- Resource/data source used
- Relevant config snippet (sanitized)
- Full error output
