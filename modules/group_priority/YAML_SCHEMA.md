# YAML Configuration Schema Reference

## Overview

Policy and extension orderings can be defined in multiple locations:

1. **Group priority YAML files** (`config/group_priority/*.yaml`): Define `default_policy_ordering`, `default_extension_ordering`, `policy_orderings`, and `extension_orderings`
2. **Policy YAML files** (`config/policies/*.yaml`): Define `policy_orderings` alongside the policies they apply to
3. **Extension YAML files** (`config/extensions/*.yaml`): Define `default_extension_ordering` and `extension_orderings` alongside extensions

**Important**: Each policy or extension can only have its ordering defined in ONE file across all sources.

## Quick Reference

### Default Policy Ordering Only (Group Priority YAML)

```yaml
# config/group_priority/priority.yaml
default_policy_ordering:
  - "high-priority-group@example.com"
  - "medium-priority-group@example.com"
  - "low-priority-group@example.com"
```

### With Policy-Specific Orderings (Group Priority YAML)

```yaml
# config/group_priority/priority.yaml
default_policy_ordering:
  - "high-priority-group@example.com"
  - "medium-priority-group@example.com"
  - "low-priority-group@example.com"

policy_orderings:
  chrome.users.IncognitoMode:
    - "security-team@example.com"
    - "high-priority-group@example.com"
    - "medium-priority-group@example.com"

  chrome.users.PasswordManager:
    - "contractors@example.com"
    - "high-priority-group@example.com"
```

### With Extension Orderings (Group Priority YAML)

```yaml
# config/group_priority/priority.yaml
default_extension_ordering:
  - "security-team@example.com"
  - "engineering@example.com"
  - "all-users@example.com"

extension_orderings:
  cjpalhdlnbpafiamejdnhcphjbkeiagm:  # uBlock Origin
    - "engineering@example.com"
    - "security-team@example.com"
```

### Policy Orderings in Policy YAML Files

```yaml
# config/policies/security_policies.yaml
policies:
  incognito_security:
    group_email: "security-team@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_UNAVAILABLE"

# Define ordering alongside the policy it applies to
policy_orderings:
  chrome.users.IncognitoMode:
    - "security-team@example.com"
    - "engineering@example.com"
```

## Field Descriptions

### Top-Level Structure (Group Priority YAML)

| Field | Type | Required | Description |
| ------- | ------ | ---------- | ------------- |
| `default_policy_ordering` | list | Yes | Default priority ordering of groups (by email) applied to all policies |
| `policy_orderings` | map | No | Policy-specific orderings that override the default for specific policies |
| `default_extension_ordering` | list | No | Default priority ordering of groups (by email) applied to all extensions |
| `extension_orderings` | map | No | Extension-specific orderings that override the default for specific extensions |

### Top-Level Structure (Policy YAML)

| Field | Type | Required | Description |
| ------- | ------ | ---------- | ------------- |
| `policies` | map | Yes | Map of policy configurations (see policies module documentation) |
| `policy_orderings` | map | No | Policy-specific orderings (same format as in group_priority YAML) |

### Top-Level Structure (Extension YAML)

| Field | Type | Required | Description |
| ------- | ------ | ---------- | ------------- |
| `extensions` | map | No | Map of extension configurations (see extensions module documentation) |
| `extension_groups` | map | No | Map of extension groups for bulk configuration |
| `extension_orderings` | map | No | Extension-specific orderings (same format as in group_priority YAML) |

**Note**: `default_extension_ordering` must be defined in group_priority YAML files, not in extension YAML files.

### Default Policy Ordering

The `default_policy_ordering` field is a list of group email addresses in priority order. Groups earlier in the list have **higher priority** and their policy settings will override those of groups later in the list.

```yaml
default_policy_ordering:
  - "highest-priority@example.com"   # Highest priority (wins conflicts)
  - "medium-priority@example.com"    # Medium priority
  - "lowest-priority@example.com"    # Lowest priority (overridden by others)
```

### Default Extension Ordering

The `default_extension_ordering` field works the same way as `default_policy_ordering` but applies to extensions. It must be defined in group_priority YAML files only (not in extension YAML files).

```yaml
default_extension_ordering:
  - "security-team@example.com"      # Highest priority
  - "engineering@example.com"
  - "all-users@example.com"          # Lowest priority
```

### Policy Orderings

The `policy_orderings` field is a map where:

- **Key**: The full policy schema name (e.g., `chrome.users.IncognitoMode`)
- **Value**: A list of group emails in priority order for that specific policy

```yaml
policy_orderings:
  chrome.users.PolicyName:
    - "group1@example.com"
    - "group2@example.com"
```

## How Priority Ordering Works

### Priority Resolution

When multiple groups have the same Chrome policy applied with different values:

1. **Higher priority group wins**: Groups earlier in the ordering list take precedence
2. **Policy-specific ordering**: If defined, overrides the default ordering for that policy
3. **Automatic filtering**: Only groups that actually have the policy applied are included in the API call

### Example Scenario

Given this configuration:

```yaml
default_policy_ordering:
  - "executives@example.com"
  - "engineering@example.com"
  - "contractors@example.com"

policy_orderings:
  chrome.users.IncognitoMode:
    - "security-team@example.com"
    - "engineering@example.com"
```

And these policies:

```yaml
policies:
  exec_incognito:
    group_email: "executives@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_AVAILABLE"

  eng_incognito:
    group_email: "engineering@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_UNAVAILABLE"
```

The resulting ordering for `chrome.users.IncognitoMode` would be:

1. `engineering@example.com` (from policy_orderings, executives not in this ordering)

Since `executives@example.com` is not in the policy-specific ordering, **this would fail validation**. You must include all groups that have the policy in either the default_policy_ordering or the policy-specific ordering.

## Validation Rules

The module enforces these validation rules during `terraform plan`:

### Rule 1: Single Default Ordering Definition

The `default_policy_ordering` field must only be defined in **one group_priority YAML file**. If multiple files define it, validation fails. Note: `default_policy_ordering` can only be defined in group_priority YAML files, not in policy YAML files.

**Error if violated:**

``` text
Multiple YAML files define 'default_policy_ordering'. Only one file should define the default ordering. Found in: priority.yaml, other_priority.yaml
```

**Solution:** Keep `default_policy_ordering` in a single group_priority file (e.g., `config/group_priority/priority.yaml`).

### Rule 2: Single Policy Ordering Definition Per Policy

Each policy can only have its ordering defined in **one file** across all sources (group_priority YAML + policy YAML files). If the same policy has ordering defined in multiple files, validation fails.

**Error if violated:**

``` text
Policy 'chrome.users.IncognitoMode' has ordering defined in multiple files: group_priority/priority.yaml, policies/security_policies.yaml. Each policy can only have one ordering definition.
```

**Solution:** Remove the duplicate ordering from one of the files. Choose to keep it either:

- In the group_priority YAML file (centralized management approach), OR
- In the policy YAML file (co-located with the policy definition)

### Rule 3: All Groups Must Be in Ordering

Every group that has a policy applied must be present in either:

- The `default_policy_ordering` list, OR
- The `policy_orderings` list for that specific policy (from any source)

**Error if violated:**

``` text
Group 'missing-group@example.com' has policy 'chrome.users.IncognitoMode' applied but is not present in the ordering definition
```

### Rule 4: Ordering Only Applied When Needed

Priority ordering is only created for policies that have **2 or more groups** with the policy applied. Single-group policies don't need ordering.

## Advanced Examples

### Complete Configuration (Group Priority YAML)

```yaml
# config/group_priority/priority.yaml

# Default ordering applies to all policies unless overridden
default_policy_ordering:
  - "security-admins@example.com"      # Highest priority
  - "executives@example.com"
  - "engineering@example.com"
  - "sales@example.com"
  - "contractors@example.com"          # Lowest priority

# Override ordering for specific policies
# Note: These can also be defined in policy YAML files instead
policy_orderings:
  # Security team has highest priority for password policies
  chrome.users.PasswordManager:
    - "security-admins@example.com"
    - "compliance@example.com"
    - "engineering@example.com"
    - "contractors@example.com"

  # Engineering has priority for developer tools
  chrome.users.DeveloperToolsAvailability:
    - "engineering@example.com"
    - "security-admins@example.com"
    - "contractors@example.com"
```

### Co-located Configuration (Policy YAML)

Define policy orderings alongside the policies they apply to:

```yaml
# config/policies/incognito_policies.yaml

policies:
  exec_incognito:
    group_email: "executives@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_AVAILABLE"

  contractor_incognito:
    group_email: "contractors@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_UNAVAILABLE"

  eng_incognito:
    group_email: "engineering@example.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "INCOGNITO_MODE_AVAILABILITY_ENUM_UNAVAILABLE"

# Ordering defined here, alongside the policies
policy_orderings:
  chrome.users.IncognitoMode:
    - "contractors@example.com"       # Most restrictive first
    - "engineering@example.com"
    - "executives@example.com"
```

### Multiple YAML Files (Split Configuration)

You can split configuration across multiple group_priority files:

**config/group_priority/priority_default.yaml:**

```yaml
default_policy_ordering:
  - "security-admins@example.com"
  - "executives@example.com"
  - "engineering@example.com"
```

**config/group_priority/priority_security_policies.yaml:**

```yaml
policy_orderings:
  chrome.users.PasswordManager:
    - "security-admins@example.com"
    - "engineering@example.com"
```

And also define policy_orderings in policy files:

**config/policies/developer_policies.yaml:**

```yaml
policies:
  dev_tools_eng:
    group_email: "engineering@example.com"
    policies:
      - schema_name: "chrome.users.DeveloperToolsAvailability"
        schema_values:
          developerToolsAvailability: "DEVELOPER_TOOLS_AVAILABILITY_ENUM_ALLOWED"

policy_orderings:
  chrome.users.DeveloperToolsAvailability:
    - "engineering@example.com"
    - "security-admins@example.com"
```

All files are merged together, with validation ensuring no duplicate policy orderings exist.

## Tips

1. **Start with default_policy_ordering**: Define a sensible default that works for most policies
2. **Use policy_orderings sparingly**: Only override when a specific policy needs different priority
3. **Security groups first**: Generally, security/compliance groups should have highest priority
4. **Include all groups**: Make sure every group with policies is in at least one ordering
5. **Review validation errors**: The module will tell you exactly which groups are missing
6. **Choose one location per policy**: Decide whether to define policy orderings centrally (group_priority YAML) or co-located (policy YAML), but not both
7. **Co-locate related orderings**: If a policy has complex ordering requirements, consider defining it alongside the policy definition for better maintainability
8. **Keep default_policy_ordering in group_priority**: The `default_policy_ordering` must always be in a group_priority YAML file

## References

- [Chrome Policy Group Priority Ordering API](https://developers.google.com/chrome/policy/reference/rest/v1/customers.policies.groups/updateGroupPriorityOrdering)
- [Understanding Policy Precedence](https://support.google.com/chrome/a/answer/9037717)
