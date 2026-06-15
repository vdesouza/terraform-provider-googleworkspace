# Variables Module

This module parses YAML files from a designated directory to provide reusable variables that can be referenced across other configuration files (policies, extensions, groups, group_priority).

## Overview

Variables help reduce duplication in your YAML configurations. Define a value once (like a list of emails) and reference it anywhere using `{variable_name}` syntax.

## Usage

### 1. Define Variables

Create YAML files in `config/variables/`:

```yaml
# config/variables/email_lists.yaml

# Lists of emails
admin_emails:
  - "admin1@company.com"
  - "admin2@company.com"
  - "admin3@company.com"

security_team_emails:
  - "security-lead@company.com"
  - "security-analyst@company.com"

# Single values
security_group: "security-team@company.com"
engineering_group: "engineering@company.com"
contractors_ou: "/Contractors"
```

### 2. Reference Variables

Use `{variable_name}` syntax in other YAML files:

```yaml
# config/policies/security_policies.yaml
policies:
  incognito_disabled:
    group_email: "{security_group}"  # Resolves to "security-team@company.com"
    policies:
      - schema_name: "chrome.users.IncognitoMode"
        schema_values:
          incognitoModeAvailability: "DISABLED"

# config/group_priority/priority.yaml
default_policy_ordering: "{admin_emails}"  # Resolves to the list of admin emails

policy_orderings:
  chrome.users.IncognitoMode: "{security_team_emails}"
```

## Variable Syntax

- **Exact reference**: `{variable_name}` - The entire field value is replaced
- **Type preservation**: If the variable is a list, the field becomes a list

### Examples

| YAML | Variable Value | Result |
|------|---------------|--------|
| `group_email: "{security_group}"` | `"security@co.com"` | `group_email: "security@co.com"` |
| `default_ordering: "{admin_emails}"` | `["a@co.com", "b@co.com"]` | `default_ordering: ["a@co.com", "b@co.com"]` |
| `members: ["{admins}", "user@co.com"]` | `admins: ["a@co.com", "b@co.com"]` | `members: ["a@co.com", "b@co.com", "user@co.com"]` |

## Supported Fields

Variables can be used in these fields:

### Policies Module
- `group_email`
- `group_key`
- `ou_path`
- `ou_key`
- `policy_orderings` lists

### Extensions Module
- `group_email`
- `group_key`
- `ou_path`
- `ou_key`

### Groups Module
- `email`
- `user_members` list
- `user_managers` list
- `user_owners` list
- `group_members` list

### Group Priority Module
- `default_policy_ordering` list
- `default_extension_ordering` list
- `policy_orderings` lists
- `extension_orderings` lists

## Directory Structure

Variables can be organized in subdirectories:

```
config/variables/
├── email_lists.yaml
├── ou_paths.yaml
└── teams/
    ├── engineering.yaml
    └── security.yaml
```

All YAML files in the directory (recursively) are parsed and merged.

## Validation

The module validates:

- **No duplicate variables**: Each variable name must be unique across all files
- Error message shows which files define duplicates

## Inputs

| Name | Description | Type | Required |
|------|-------------|------|----------|
| `yaml_config_path` | Path to the directory containing variable YAML files | `string` | Yes |

## Outputs

| Name | Description |
|------|-------------|
| `variables_map` | Map of all variables (name => value) |
| `variable_lookup` | Lookup map for resolution ({name} => value) |
| `variable_names` | List of all defined variable names |
| `variables_by_file` | Variables grouped by source file (for debugging) |
