# Group Priority Module

This Terraform module manages Chrome Policy group priority ordering from YAML configuration files. It determines the order in which group-based policies and extensions are applied when multiple groups have the same policy/extension with different values.

## Overview

When multiple groups have the same Chrome policy or extension applied with conflicting values, Google Workspace uses priority ordering to determine which group's settings take precedence. Groups with higher priority (earlier in the list) override settings from groups with lower priority.

This module:

- Reads priority ordering configuration from YAML files
- Supports a default ordering for all policies (`default_policy_ordering`)
- Supports a default ordering for all extensions (`default_extension_ordering`)
- Supports policy-specific orderings that override the default
- Supports extension-specific orderings that override the default
- Accepts policy orderings from external sources (e.g., policy YAML files)
- Accepts extension orderings from the extensions module
- Creates group priority ordering resources for both policies and extensions
- Validates that all groups with policies/extensions are included in orderings
- Validates that each policy/extension ordering is only defined once across all sources
- Automatically filters orderings to only include groups that have each policy applied
- Supports **unmanaged groups** in orderings (groups assigned in Google Admin but not managed by Terraform)
- Detects **API mismatches** at plan time: fails with a human-readable error if groups exist in the API ordering but are not accounted for in config

## Features

- YAML-based configuration for easy management
- Default ordering that applies to all policies (`default_policy_ordering`)
- Default ordering that applies to all extensions (`default_extension_ordering`)
- Policy-specific and extension-specific ordering overrides
- **Flexible placement**: Define orderings in group_priority YAML files OR in policy/extension YAML files
- Cross-file duplicate detection with clear error messages
- Automatic validation during `terraform plan`
- Automatic filtering of groups per policy
- Only creates ordering resources when 2+ groups have the same policy
- Looks up unmanaged ordering groups via `data.googleworkspace_group` (fails plan with clear error on typos)
- Queries the Chrome Policy API at plan time to detect groups with policies/extensions assigned in Google Admin but missing from Terraform config

## Usage

### Basic Usage (Regular User Policies)

For regular user policies (`chrome.users.*` like IncognitoMode, Wallpaper, etc.):

```hcl
module "group_priority" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/group_priority?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/group_priority"
  groups_map       = module.groups.all_groups
  policies_map     = module.policies.group_policies_detail
}
```

### App-Specific Policies

For app-specific policies (`chrome.users.apps.*`), you must also provide `org_unit_id` and `app_id`:

```hcl
module "group_priority" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/group_priority?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/group_priority"
  groups_map       = module.groups.all_groups
  policies_map     = module.policies.group_policies_detail
  org_unit_id      = "orgunits/${data.googleworkspace_org_unit.root.id}"
  app_id           = "chrome:gmbgaklkmjakoegficnlkhebmhkjfich"  # Chrome extension ID
}
```

### Full Integration Example

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

  yaml_config_paths = ["${path.module}/config/policies"]
  groups_map        = module.groups.all_groups
  assets_map        = module.assets.assets_map
}

module "extensions" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/extensions?ref=v1.4.0"

  yaml_config_paths = ["${path.module}/config/extensions"]
  groups_map        = module.groups.all_groups
}

module "group_priority" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/group_priority?ref=v1.4.0"

  yaml_config_path                      = "${path.module}/config/group_priority"
  groups_map                            = module.groups.all_groups

  # Policy orderings from policies module
  policies_map                          = module.policies.group_policies_detail
  external_policy_orderings             = module.policies.policy_orderings
  external_policy_orderings_with_source = module.policies.policy_orderings_with_source

  # Extension orderings from extensions module
  # Note: default_extension_ordering must be defined in group_priority YAML files only
  extensions_map                           = module.extensions.group_extensions_detail
  external_extension_orderings             = module.extensions.extension_orderings
  external_extension_orderings_with_source = module.extensions.extension_orderings_with_source
}
```

## YAML Configuration

Orderings can be defined in multiple locations, with specific rules:

1. **Group priority YAML files** (`config/group_priority/*.yaml`): The **only** location for `default_policy_ordering` and `default_extension_ordering`. Can also define `policy_orderings` and `extension_orderings`.
2. **Policy YAML files** (`config/policies/*.yaml`): Define `policy_orderings` alongside the policies they apply to
3. **Extension YAML files** (`config/extensions/*.yaml`): Define `extension_orderings` alongside extensions

**Important**:
- `default_policy_ordering` and `default_extension_ordering` must be defined in group_priority YAML files only
- Each policy or extension can only have its specific ordering defined in ONE file across all sources
- If a policy/extension ordering is defined in multiple files, validation will fail with a clear error message

See [YAML_SCHEMA.md](YAML_SCHEMA.md) for full schema details.

### Example Configuration (Group Priority YAML)

Define `default_policy_ordering` and `default_extension_ordering` in the group priority directory:

```yaml
# config/group_priority/priority.yaml

# ============================================================================
# POLICY ORDERINGS
# ============================================================================

# Default ordering applies to all policies (must only be defined once)
default_policy_ordering:
  - "security-admins@example.com"
  - "executives@example.com"
  - "engineering@example.com"
  - "contractors@example.com"

# Override ordering for specific policies (optional)
# Can also be defined in policy YAML files
policy_orderings:
  chrome.users.PasswordManager:
    - "security-admins@example.com"
    - "compliance@example.com"
    - "engineering@example.com"

# ============================================================================
# EXTENSION ORDERINGS
# ============================================================================

# Default ordering applies to all extensions (must only be defined once)
# Can also be defined in extension YAML files, but not both
default_extension_ordering:
  - "security-team@example.com"
  - "engineering@example.com"
  - "all-users@example.com"

# Override ordering for specific extensions (optional)
# Key is the extension ID (without chrome:/android:/web: prefix)
extension_orderings:
  cjpalhdlnbpafiamejdnhcphjbkeiagm:  # uBlock Origin
    - "engineering@example.com"
    - "security-team@example.com"
```

### Example Configuration (Policy YAML)

Define `policy_orderings` alongside the policies:

```yaml
# config/policies/incognito_policies.yaml

policies:
  eng_incognito:
    group_email: "engineering@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_UNAVAILABLE"

  exec_incognito:
    group_email: "executives@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_AVAILABLE"

# Define ordering alongside the policies it applies to
policy_orderings:
  chrome.users.IncognitoMode:
    - "security-admins@example.com"
    - "engineering@example.com"
    - "executives@example.com"
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| yaml_config_path | Path to directory containing YAML group priority configuration files | string | - | yes |
| groups_map | Map of group emails to group details (id, email) | map(object) | - | yes |
| policies_map | Map of policy keys to policy details for determining which groups have which policies | map(object) | - | yes |
| external_policy_orderings | Map of policy-specific orderings from external sources (e.g., policy YAML files) | map(list(string)) | {} | no |
| external_policy_orderings_with_source | Map of policy orderings with source file info for duplicate validation | map(object) | {} | no |
| extensions_map | Map of extension keys to extension details (from extensions module) for determining which groups have which extensions | map(object) | {} | no |
| external_extension_orderings | Map of extension-specific orderings from extensions module | map(list(string)) | {} | no |
| external_extension_orderings_with_source | Map of extension orderings with source file info for duplicate validation | map(object) | {} | no |

## Outputs

| Name | Description |
|------|-------------|
| policy_orderings | Map of policy schemas to their group priority orderings |
| policies_with_ordering | List of policy schemas that have group priority ordering configured |
| policies_without_ordering | List of policy schemas that don't need ordering (single group or no groups) |
| default_policy_ordering | The default policy group ordering (by email) |
| default_policy_ordering_source_file | The YAML file that defines the default policy ordering |
| policy_specific_orderings | Map of policy-specific orderings defined in YAML (merged from all sources) |
| local_policy_orderings | Map of policy orderings defined in group_priority YAML files |
| external_policy_orderings | Map of policy orderings from external sources (policy YAML files) |
| policy_orderings_with_source | Map of policy orderings with their source file information |
| duplicate_policy_orderings | Map of policy schemas with duplicate ordering definitions (should be empty) |
| default_extension_ordering | The default extension group ordering (by email) |
| default_extension_ordering_source_file | The YAML file that defines the default extension ordering |
| extension_orderings | Map of extension-specific orderings (merged from all sources) |
| extension_orderings_with_source | Map of extension orderings with their source file information |
| extensions_with_ordering | List of extension app_ids that have group priority ordering configured |
| extensions_without_ordering | List of extension app_ids that don't need ordering (single group or no groups) |
| validation_errors | List of validation errors (empty if all validations pass) |

## How It Works

### 1. Configuration Loading

The module reads configuration from multiple sources:

**From group_priority YAML files:**
- `default_policy_ordering`: List of group emails in priority order for policies
- `default_extension_ordering`: List of group emails in priority order for extensions
- `policy_orderings`: Map of policy schema -> custom ordering
- `extension_orderings`: Map of extension ID -> custom ordering

**From external sources (policy YAML files via module inputs):**
- `external_policy_orderings`: Policy orderings defined alongside policies

All `policy_orderings` from both sources are merged together. Extension orderings from both group_priority YAML files and extension YAML files (via module inputs) are merged and used to create extension ordering resources.

### 2. Group Resolution

For each policy schema managed by the policies module:

- Identifies all groups that have the policy applied
- Determines which ordering to use (policy-specific or default)
- Includes all ordering emails that resolve to a known group ID (managed or unmanaged)

**Unmanaged groups**: If an email appears in an ordering config but is not managed by Terraform, the module looks it up via `data.googleworkspace_group` by email. If the email doesn't exist in Google Workspace, the plan fails with a clear error (catches typos). This allows groups that are assigned policies directly in Google Admin to be included in Terraform's ordering.

### 3. API Mismatch Validation

At plan time, the module queries the Chrome Policy API to get the current group ordering for each policy/extension that needs ordering. It compares this against what Terraform will send and fails with a human-readable error if there are groups in the API that are not accounted for in your config:

```
Error: Group priority ordering validation failed:
Policy 'chrome.users.ExternalStorage': Group ID '12345678901' is assigned this policy in Google Admin but is not in the Terraform ordering. Look up this group ID in Google Admin, then add its email to the ordering in your YAML config (default_policy_ordering or policy_orderings) or remove its policy assignment in Google Admin Console.
```

This catches the case where someone has manually assigned a policy to a group in Google Admin, which would cause the `UpdateGroupPriorityOrdering` API to return HTTP 400. The error shows the numeric group ID; look it up in the Google Admin console (Directory → Groups) to find the email.

**Edge cases**:
- New environment (no policies assigned yet): API returns 400 → `exists=false` → no API validation → correct behavior
- All groups accounted for: API matches Terraform → no errors
- Unmanaged group added to ordering YAML: Resolved via data source → included in ordering → API happy

### 4. Validation

Before creating resources, the module validates:

- Only one YAML file defines `default_policy_ordering` (prevents conflicts from multiple files)
- Only one source defines `default_extension_ordering` (either group_priority or extension YAML files)
- Each policy ordering is only defined once across all sources (group_priority YAML + policy YAML files)
- Each extension ordering is only defined once across all sources (group_priority YAML + extension YAML files)
- All Terraform-managed groups with policies are present in at least one ordering
- All groups in the API ordering are accounted for in Terraform's ordering config
- Fails `terraform plan` with clear error messages if validation fails

### 4. Resource Creation

Creates `googleworkspace_chrome_policy_group_priority_ordering` resources for each policy that has 2+ groups.

## Validation Errors

### Duplicate Default Policy Ordering

If multiple YAML files define `default_policy_ordering`, you'll see:

```
Error: Group priority ordering validation failed:
Multiple YAML files define 'default_policy_ordering'. Only one file should define the default ordering. Found in: priority.yaml, other_priority.yaml
```

**Solution**: Remove `default_policy_ordering` from all but one YAML file. Keep it in a single file (e.g., `priority.yaml`) and use other files only for `policy_orderings`.

### Duplicate Default Extension Ordering

If `default_extension_ordering` is defined in both group_priority and extension YAML files, you'll see:

```
Error: default_extension_ordering is defined in both group_priority YAML and extensions YAML files.
```

**Solution**: Remove `default_extension_ordering` from one source. Define it either in group_priority YAML (centralized) or in extension YAML (co-located with extensions).

### Duplicate Policy Ordering

If the same policy has ordering defined in multiple files (across group_priority and policy YAML files), you'll see:

```
Error: Group priority ordering validation failed:
Policy 'chrome.users.IncognitoMode' has ordering defined in multiple files: group_priority/priority.yaml, policies/security_policies.yaml. Each policy can only have one ordering definition.
```

**Solution**: Remove the duplicate policy ordering from one of the files. Choose either:
- Keep the ordering in the group_priority YAML file (centralized management), OR
- Keep the ordering in the policy YAML file (co-located with the policy definition)

### Missing Group in Ordering

If a Terraform-managed group has a policy applied but is not in the ordering, you'll see:

```
Error: Group priority ordering validation failed:
Group 'missing-group@example.com' has policy 'chrome.users.IncognitoMode' applied but is not present in the ordering definition
```

**Solution**: Add the group to either `default_policy_ordering` or the policy-specific ordering in `policy_orderings`.

### API Mismatch: Unmanaged Group Not in Ordering

If a group has the policy assigned in Google Admin (outside Terraform) but is not in your ordering config:

```
Error: Group priority ordering validation failed:
Policy 'chrome.users.ExternalStorage': Group ID '12345678901' is assigned this policy in Google Admin but is not in the Terraform ordering. Look up this group ID in Google Admin, then add its email to the ordering in your YAML config (default_policy_ordering or policy_orderings) or remove its policy assignment in Google Admin Console.
```

**Solution**: Either:
1. Look up the group ID `12345678901` in Google Admin (Directory → Groups), then add its email to `default_policy_ordering` or `policy_orderings` in YAML, **or**
2. Remove the policy assignment from this group in Google Admin Console.

## Directory Structure

```text
config/group_priority/
├── priority.yaml           # Main priority configuration
└── policy_overrides.yaml   # Optional: policy-specific overrides
```

## Requirements

- Terraform >= 1.3
- googleworkspace provider >= 1.3.13
- null provider (for validation)

## Resources Created

- `googleworkspace_chrome_policy_group_priority_ordering.ordering`: One per policy schema with 2+ groups
- `googleworkspace_chrome_policy_group_priority_ordering.extension_ordering`: One per extension app_id with 2+ groups
- `null_resource.validate_orderings`: Validation resource (only created if errors exist)

## Notes

- Priority ordering only matters when multiple groups have the same policy/extension with different values
- Groups earlier in the list have higher priority (win conflicts)
- The module automatically handles cases where not all groups in the ordering have a particular policy/extension
- Policy-specific orderings completely override the default for that policy
- Extension-specific orderings completely override the default for that extension
- `policy_orderings` can be defined in either group_priority YAML files or policy YAML files
- `extension_orderings` can be defined in either group_priority YAML files or extension YAML files
- Each policy/extension can only have ONE ordering definition across all files (enforced by validation)
- `default_policy_ordering` must always be defined in a group_priority YAML file (not in policy YAML files)
- `default_extension_ordering` must always be defined in a group_priority YAML file (not in extension YAML files)
- Extension-specific orderings (`extension_orderings`) can be defined in either group_priority OR extension YAML files
- Extension orderings from extension YAML files are passed to this module and merged with local orderings

## References

- [Chrome Policy Group Priority Ordering API](https://developers.google.com/chrome/policy/reference/rest/v1/customers.policies.groups/updateGroupPriorityOrdering)
- [Understanding Policy Precedence](https://support.google.com/chrome/a/answer/9037717)
- [Terraform Google Workspace Provider](https://registry.terraform.io/providers/vdesouza/googleworkspace)
