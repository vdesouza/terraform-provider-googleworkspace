# Companion Modules

These Terraform modules ship from the [`terraform-provider-googleworkspace`](https://github.com/vdesouza/terraform-provider-googleworkspace) repository as a higher-level, YAML-driven layer on top of the provider. They turn declarative YAML configuration into Google Workspace groups, Chrome policies, Chrome extensions, and policy ordering resources.

The modules are not published to the Terraform Module Registry. They are consumed via a Git source pinned to a provider release tag:

```hcl
module "chrome_policies" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/policies?ref=v1.4.0"
  # ...
}
```

Use the same `?ref=` tag across all modules in a single configuration so their schemas stay in sync. When upgrading the provider, bump the `?ref=` tag in lockstep.

## Modules

| Module | Manages | Description |
|---|---|---|
| [`variables`](variables/) | (none) | Parses YAML variable files and exports a `variables_map` consumed by every other module. |
| [`groups`](groups/) | `googleworkspace_group`, `googleworkspace_group_dynamic`, `googleworkspace_group_member` | Static and dynamic Google Workspace groups defined in YAML. |
| [`assets`](assets/) | `googleworkspace_chrome_policy_file` | Uploads files (wallpapers, avatars, terms-of-service docs) to Chrome Policy storage and exposes their `download_uri` for use in policies. |
| [`extensions`](extensions/) | `googleworkspace_chrome_group_policy`, `googleworkspace_chrome_policy` | Chrome extensions, Android apps, and web apps deployed to groups or org units. |
| [`policies`](policies/) | `googleworkspace_chrome_group_policy`, `googleworkspace_chrome_policy` | Chrome policies for groups and org units, with asset reference resolution. |
| [`group_priority`](group_priority/) | `googleworkspace_chrome_policy_group_priority_ordering` | Resolves ordering when multiple groups assign overlapping policies or extensions. |

## Dependency

- `variables` feeds `variables_map` to every module that takes one.
- `groups` produces `all_groups` consumed by `policies`, `extensions`, and `group_priority`.
- `assets` produces `assets_map` consumed by `policies` for `${asset.<key>/<field>}` substitution.
- `policies` and `extensions` produce ordering inputs consumed by `group_priority`.

## Quick Start

1. Authenticate the provider in your root module (see the top-level README).
2. Pick the modules you need from the table above. Most users will start with `variables` + `groups` + `policies`.
3. Drop the `module "x" { source = "git::...//modules/<name>?ref=v1.4.0" }` blocks into your root config and wire outputs through inputs as the dependency graph shows.
4. Author YAML configuration files matching each module's `YAML_SCHEMA.md`.
5. `terraform init && terraform plan`.

