# Chrome Policies Module

This Terraform module manages Chrome policies in Google Workspace using YAML configuration files.

## Overview

The module reads YAML configuration files that define Chrome policies to be applied to either groups or organizational units (OUs). It supports:

- **Group-based policies**: Apply policies to specific Google Workspace groups
- **OU-based policies**: Apply policies to organizational units
- Multiple policies per target (group or OU)
- Policy configuration via YAML files
- Group reference by email or by key (from groups module)
- OU reference by path or by key (from ou_map)
- Additional targeting parameters
- Automatic validation of target references

## Usage

### Basic Usage

```hcl
module "policies" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/policies?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/policies"
}
```

### With Groups Module Integration

```hcl
module "groups" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/groups?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/groups"
}

module "policies" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/policies?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/policies"
  groups_map       = module.groups.all_groups
}
```

### With Groups and Assets Module Integration

```hcl
module "groups" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/groups?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/groups"
}

module "assets" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/assets?ref=v1.4.0"

  yaml_config_path   = "${path.module}/config/assets"
  assets_source_path = "${path.module}/config/assets"
}

module "policies" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/policies?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/policies"
  groups_map       = module.groups.all_groups
  assets_map       = module.assets.assets_map
}
```

## YAML Configuration

Place your YAML files in the `config/policies` directory. See [YAML_SCHEMA.md](./YAML_SCHEMA.md) for detailed schema documentation.

### Group-Based Policy Example

```yaml
policies:
  engineering_security:
    group_email: "engineering@example.com"
    policies:
      - schema_name: "chrome.users.PasswordManagerEnabled"
        schema_values:
          passwordManagerEnabled: false
```

### OU-Based Policy Example

```yaml
policies:
  engineering_ou_security:
    ou_path: "Engineering/Development"
    policies:
      - schema_name: "chrome.users.ChromeOsLockOnIdleSuspend"
        schema_values:
          chromeOsLockOnIdleSuspend: true
```

### With Group Module Reference

```yaml
policies:
  contractor_restrictions:
    group_key: "contractors_group"  # References group from groups module
    policies:
      - schema_name: "chrome.users.DownloadRestrictions"
        schema_values:
          downloadRestrictions: 3
      - schema_name: "chrome.users.IncognitoModeAvailability"
        schema_values:
          incognitoModeAvailability: 1
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| yaml_config_path | Path to directory containing YAML policy configuration files | string | - | yes |
| groups_map | Map of group keys to group details (from groups module) | map(object) | {} | no |
| ou_map | Map of OU keys to OU details for resolving ou_key references | map(object) | {} | no |
| assets_map | Map of asset keys to asset details (from assets module) for resolving asset references | map(object) | {} | no |

## Outputs

| Name | Description |
|------|-------------|
| all_policies | Map of all policies created (both group and OU-based) |
| group_policies | Map of all group-based policies created |
| ou_policies | Map of all OU-based policies created |
| policy_count | Total number of policies created (group + OU) |
| group_policy_count | Number of group-based policies created |
| ou_policy_count | Number of OU-based policies created |
| policies_by_group | Map of group IDs to their associated policies |
| policies_by_ou | Map of OU IDs to their associated policies |

## Features

### Target Reference Methods

The module supports policies for both groups and organizational units:

#### Group-Based Policies

1. **Direct Email Reference**: Use `group_email` to directly specify the group email
2. **Group Module Reference**: Use `group_key` to reference a group from the groups module

When using `group_key`, the module will:
- Look up the group ID from the groups module output
- Create dependencies to ensure groups are created before policies
- Validate that the referenced group exists

#### OU-Based Policies

1. **Direct Path Reference**: Use `ou_path` to directly specify the organizational unit path
2. **OU Map Reference**: Use `ou_key` to reference an OU from the ou_map variable

When using `ou_path`, the module will:
- Use a data source to look up the OU ID from the path
- Validate that the OU exists

When using `ou_key`, the module will:
- Look up the OU ID from the ou_map variable
- Create dependencies if OUs are managed in a separate module

### Validation

The module includes automatic validation:
- Ensures all policies have valid target references (group or OU)
- Fails during plan if any references cannot be resolved
- Validates that policies don't specify both group and OU targets
- Validates that policies specify at least one target (group or OU)
- Provides clear error messages for misconfigured policies

### Multiple Policies

You can apply multiple policy schemas to a single target (group or OU) by including multiple items in the `policies` list.

### Additional Targeting

Use `additional_target_keys` to add extra targeting parameters beyond the group/OU level.

### Importing Existing Policies

When adopting existing Chrome policies that are already configured in Google Admin, add `import: true` to bring them into Terraform state without re-creating them.

**Requirements:**
- Provider `vdesouza/googleworkspace` with import support
- Terraform >= 1.7 (for `import` blocks with `for_each`)

**Workflow:**

1. Add `import: true` to each policy entry in your YAML config:

```yaml
policies:
  root_CloudReporting:
    import: true
    ou_path: "/"
    policies:
      - schema_name: "chrome.users.CloudReporting"
        schema_values:
          cloudReportingEnabled: true
```

2. Run `terraform plan` to verify the imports will succeed. The provider performs strict validation: it calls the Chrome Policy API to confirm each policy is **explicitly set** on the target OU/group (not just inherited from a parent).

3. Run `terraform apply` to import the resources into state.

4. Remove `import: true` from all YAML entries. This step is required -- Terraform errors if an `import` block targets a resource already in state.

5. Run `terraform plan` to confirm zero changes (clean state).

**Validation behavior:**

| Scenario | Result |
|---|---|
| Policy explicitly set on target | Import succeeds |
| Policy inherited from parent OU | Import fails with `"not explicitly set on... (inherited from...)"` |
| Invalid schema name | Import fails with API error |
| Invalid target ID | Import fails with API error |

**Import ID format** (for manual `terraform import`):
```
<target_id>/<schema1>,<schema2>,...
```

Example:
```bash
terraform import 'module.policies.googleworkspace_chrome_policy.ou_policy["root_CloudReporting"]' \
  '03ph8a2z1shzb7m/chrome.users.CloudReporting'
```

## Examples

See `config/policies/example_policies.yaml` for more examples including:
- Group-based policies with direct email and module references
- OU-based policies with path and key references
- Multiple policies for a single target
- Complex schema values
- Additional targeting parameters
- Various policy types (security, browser, extensions, etc.)

## Requirements

- Terraform >= 1.3
- googleworkspace provider >= 1.3.13

## Resources Created

- `googleworkspace_chrome_group_policy`: Chrome group policies for each group-based configuration
- `googleworkspace_chrome_policy`: Chrome OU policies for each OU-based configuration
- `data.googleworkspace_org_unit`: Data sources for OU lookups (when using ou_path)

## Notes

- Policy schema names must be fully qualified (e.g., `chrome.users.PolicyName`)
- Schema values are automatically JSON-encoded
- The module validates all target references before creating resources
- When using `group_key`, ensure the groups module is deployed first
- When using `ou_path`, ensure the OU exists in your Google Workspace
- Each policy must specify exactly one target type (group OR OU, not both)
- When migrating a policy from OU-based to group-based, the module ensures the group policy is created before the OU policy is removed. Both briefly coexist -- the Chrome Policy API resolves conflicts in favor of group policies.

## References

- [Chrome Policy List](https://chromeenterprise.google/policies/)
- [Chrome Management API](https://developers.google.com/chrome/management)
- [Terraform Google Workspace Provider](https://registry.terraform.io/providers/vdesouza/googleworkspace)
