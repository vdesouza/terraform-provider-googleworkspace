# Extensions Module

This Terraform module manages Chrome extensions, Android apps, and web apps from YAML configuration files with a simplified, human-readable schema.

## Overview

The Extensions module makes it easy to manage Chrome extensions and apps by abstracting away the complexity of the Chrome Policy API. You simply provide:

- Extension ID (from the Chrome Web Store)
- Installation type (FORCED, ALLOWED, BLOCKED, etc.)
- Target group or OU

The module handles all the API complexity automatically.

## Features

- **Multiple app types**: Supports Chrome extensions, Android apps, and web apps
- **Group-based and OU-based**: Deploy extensions to specific groups or entire organizational units
- **Extension groups**: Configure multiple extensions with shared settings
- **Common install types**: Simple `install_type` field instead of complex schema values
- **Advanced configurations**: Optional configurations list for complex setups
- **Group ordering**: Define priority order when multiple groups have the same extension
- **YAML-based configuration**: Easy to read, write, and version control
- **Automatic validation**: Validates all configurations during `terraform plan`

## Usage

### Basic Usage

```hcl
module "extensions" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/extensions?ref=v1.4.0"

  yaml_config_paths = ["${path.module}/config/extensions"]
  groups_map        = module.groups.all_groups
}
```

### Full Integration Example

Extensions can be defined in any YAML file. This allows you to keep related configurations together. Extension orderings are passed to the `group_priority` module.

```hcl
module "groups" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/groups?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/groups"
}

module "extensions" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/extensions?ref=v1.4.0"

  # Extensions can be defined in any YAML file in these directories
  yaml_config_paths = [
    "${path.module}/config/extensions",
    "${path.module}/config/policies",  # Also read extensions from policy files
  ]
  groups_map = module.groups.all_groups
}

module "group_priority" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/group_priority?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/group_priority"
  groups_map       = module.groups.all_groups
  policies_map     = module.policies.group_policies_detail

  # Extension orderings from extensions module
  extensions_map                           = module.extensions.group_extensions_detail
  external_extension_orderings             = module.extensions.extension_orderings
  external_extension_orderings_with_source = module.extensions.extension_orderings_with_source
}
```

## YAML Configuration

Extensions can be defined in any YAML file - either in a dedicated extensions directory or alongside policies. See [YAML_SCHEMA.md](YAML_SCHEMA.md) for full schema details.

### Mixed Configuration Example

You can define both policies and extensions in the same YAML file:

```yaml
# config/policies/security.yaml - Security policies AND extensions together

policies:
  security_browser_settings:
    group_key: "all_users"
    policies:
      - schema_name: "chrome.users.SafeBrowsing"
        schema_values:
          safeBrowsingEnabled: true

extensions:
  ublock_origin_all:
    extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
    group_key: "all_users"
    install_type: "FORCED"
```

### Extension Groups (Bulk Configuration)

Use `extension_groups` to configure multiple extensions with shared settings:

```yaml
# config/extensions/developer_tools.yaml

extension_groups:
  # Allow developer tools for engineering team
  developer_allowlist:
    group_key: "engineering"
    install_type: "ALLOWED"
    extensions:
      - "bcjindcccaagfpapjjmafapmmgkkhgoa"  # JSON Formatter
      - "fmkadmapgofadopljbjfkapdkoienihi"  # React DevTools
      - "lmhkpmbekcpmknklioeibfkpmmfibljd"  # Redux DevTools

  # Block unwanted extensions for everyone
  blocked_extensions:
    group_key: "all_users"
    install_type: "BLOCKED"
    extensions:
      - "suspiciousextension123"
      - "cryptominerextension456"
```

### Extension Groups with Variables

The `extensions` list in `extension_groups` supports variable references. This allows you to define reusable lists of extension IDs in your variables config:

```yaml
# config/variables/extension_lists.yaml
developer_tools_list:
  - "bcjindcccaagfpapjjmafapmmgkkhgoa"  # JSON Formatter
  - "fmkadmapgofadopljbjfkapdkoienihi"  # React DevTools
  - "lmhkpmbekcpmknklioeibfkpmmfibljd"  # Redux DevTools
```

```yaml
# config/extensions/developer_tools.yaml
extension_groups:
  # Use a variable for the entire extensions list
  developer_allowlist:
    group_key: "engineering"
    install_type: "ALLOWED"
    extensions: "{developer_tools_list}"  # Resolves to the list above
```

You can also mix direct extension IDs with variable references in the same list:

```yaml
extension_groups:
  mixed_extensions:
    group_key: "engineering"
    install_type: "ALLOWED"
    extensions:
      - "someextensionid123"        # Direct ID
      - "{developer_tools_list}"    # Variable reference (expands to multiple IDs)
      - "anotherextensionid456"     # Direct ID
```

### Configurations with Variables

The `configurations` field also supports variable references. This allows you to define reusable configuration blocks:

```yaml
# config/variables/configs.yaml
blocked_dangerous_permissions:
  - blockedPermissions:
      - "webRequest"
      - "proxy"
      - "debugger"
```

```yaml
# config/extensions/security.yaml
extension_groups:
  security_extensions:
    group_key: "all_users"
    install_type: "FORCED"
    configurations: "{blocked_dangerous_permissions}"  # Resolves to list of configs
    extensions:
      - "cjpalhdlnbpafiamejdnhcphjbkeiagm"  # uBlock Origin
      - "gcbommkclmclpchllfjekcdonpmejbdp"  # HTTPS Everywhere
```

This is useful for applying the same permissions or managed configuration to multiple extension groups.

### Individual Extensions

Use `extensions` when each extension needs unique settings:

```yaml
# config/extensions/security_extensions.yaml

extensions:
  # Force install uBlock Origin for all engineers
  ublock_engineering:
    extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
    group_email: "engineering@example.com"
    install_type: "FORCED"

  # Allow Google Translate for everyone
  google_translate:
    extension_id: "aapbdbdomjkkjkaonfhkkikfgjllcleb"
    group_key: "all_users"
    install_type: "ALLOWED"
```

## Installation Types

| Install Type | Description |
| ------------- | ------------- |
| `FORCED` | Force install the extension (users cannot remove it) |
| `ALLOWED` | Allow users to install the extension if they want |
| `BLOCKED` | Block installation of the extension |
| `FORCED_AND_PIN_TO_TOOLBAR` | Force install and pin to the browser toolbar |
| `NORMAL` | Force install but allow users to disable (Chrome extensions only) |
| `NORMAL_AND_PIN_TO_TOOLBAR` | Force install, pin to toolbar, but allow users to disable (Chrome extensions only) |
| `REMOVE` | Block installation and remove from devices (Chrome extensions only) |

## App Types

The module automatically detects the app type based on which field you use:

### Chrome Extensions

Use `extension_id` or `chrome_extension_id`:

```yaml
extensions:
  my_extension:
    extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
    group_key: "engineers"
    install_type: "FORCED"
```

### Android Apps

Use `android_app_id`:

```yaml
extensions:
  google_keep:
    android_app_id: "com.google.android.keep"
    group_key: "chromeos_users"
    install_type: "ALLOWED"
```

### Web Apps

Use `web_app_url`:

```yaml
extensions:
  google_docs:
    web_app_url: "https://docs.google.com"
    group_key: "all_users"
    install_type: "FORCED"
```

## Advanced Configuration

For complex extension configurations, use the `configurations` field with simplified field names:

```yaml
extensions:
  tampermonkey_devs:
    extension_id: "dhdgffkkebhmkfjojejmpbldmpobfkfo"
    group_email: "developers@example.com"
    install_type: "ALLOWED"
    configurations:
      # Block specific permissions (auto-maps to PermissionsAndUrlAccess)
      - blockedPermissions:
          - "geolocation"
          - "notifications"
      # Allow other permissions
      - allowedPermissions:
          - "storage"
          - "tabs"
      # Add managed configuration (auto-maps to ManagedConfiguration)
      - managedConfiguration: |
          {
            "apiKey": "company-key",
            "serverUrl": "https://internal.example.com"
          }
```

### Supported Configuration Fields

| Field Name | Auto-maps to Schema |
| ------------ | --------------------- |
| `blockedPermissions` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `allowedPermissions` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `blockedHosts` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `allowedHosts` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `managedConfiguration` | `chrome.users.apps.ManagedConfiguration` |
| `allowAccessToKeys` | `chrome.users.apps.CertificateManagement` |

Fields that map to the same schema are automatically merged. You can also use the explicit format with `schema_name` and `schema_values` if needed:

```yaml
extensions:
  explicit_format_example:
    extension_id: "exampleextensionid123"
    group_key: "admin_users"
    install_type: "FORCED"
    configurations:
      - schema_name: "chrome.users.apps.PermissionsAndUrlAccess"
        schema_values:
          blockedPermissions:
            - "webRequest"
            - "proxy"
          allowedHosts:
            - "https://internal.example.com/*"
      - schema_name: "chrome.users.apps.ManagedConfiguration"
        schema_values:
          managedConfiguration: |
            {
              "apiKey": "company-api-key",
              "serverUrl": "https://internal.example.com"
            }
```

## Group Ordering

When multiple groups have the same extension applied with different configurations, group ordering determines which settings take precedence.

**Where orderings can be defined:**

- `default_extension_ordering`: **Only** one instance of `default_extension_ordering` and only in group priority YAML files (`config/group_priority/*.yaml`)
- `extension_orderings`: Either in extension YAML files OR group priority YAML files

Each extension can only have ONE ordering definition across all files.

### Default Ordering

The default ordering applies to all extensions and must be defined in group priority YAML files:

```yaml
# config/group_priority/priority.yaml
default_extension_ordering:
  - "security-team@example.com"    # Highest priority
  - "engineering@example.com"
  - "all-users@example.com"        # Lowest priority
```

### Extension-Specific Ordering

Override the default for specific extensions. These can be defined in extension YAML files:

```yaml
# config/extensions/my_extensions.yaml
extension_orderings:
  cjpalhdlnbpafiamejdnhcphjbkeiagm:  # uBlock Origin
    - "engineering@example.com"
    - "security-team@example.com"
    - "all-users@example.com"
```

Or in group priority YAML files:

```yaml
# config/group_priority/priority.yaml
extension_orderings:
  cjpalhdlnbpafiamejdnhcphjbkeiagm:
    - "engineering@example.com"
    - "security-team@example.com"
```

### Ordering Validation

The extensions module parses `extension_orderings` and passes them to the `group_priority` module, which:

- Creates `googleworkspace_chrome_policy_group_priority_ordering` resources for extensions
- Validates that `default_extension_ordering` is only defined in group_priority YAML files
- Validates each extension has only one ordering definition (across all sources)
- Validates all groups with extensions are included in an ordering
- Validates all groups in orderings exist in `groups_map`

## Inputs

| Name | Description | Type | Default | Required |
| ------ | ------------- | ------ | --------- | ---------- |
| yaml_config_paths | List of paths to directories containing YAML configuration files. Extensions can be defined in any YAML file with an `extensions` section. | list(string) | - | yes |
| groups_map | Map of group keys to group details for resolving group_key references | map(object) | {} | no |
| ou_map | Map of OU keys to OU details for resolving ou_key references | map(object) | {} | no |

## Outputs

| Name | Description |
| ------ | ------------- |
| group_extensions | Map of all group-based extension policies created |
| ou_extensions | Map of all OU-based extension policies created |
| all_extensions | Map of all extension policies (both group and OU-based) |
| extension_count | Total number of extension policies created |
| group_extension_count | Number of group-based extension policies created |
| ou_extension_count | Number of OU-based extension policies created |
| extensions_by_group | Map of group IDs to their associated extension policies |
| extensions_by_ou | Map of OU IDs to their associated extension policies |
| group_extensions_detail | Detailed map of group extensions for group_priority module |
| extension_orderings | Map of extension-specific orderings from YAML files |
| extension_orderings_by_file | Map of extension orderings grouped by source file |
| extension_orderings_with_source | Map of extension orderings with source file information |
| validation_errors | Map of validation errors detected (empty if no errors) |

## How It Works

### 1. Configuration Loading

The module reads all YAML files from the config directory and parses the `extensions` section.

### 2. App ID Resolution

For each extension, the module:

- Detects the app type (Chrome, Android, or Web)
- Constructs the full app_id with appropriate prefix:
  - Chrome: `chrome:extension_id`
  - Android: `android:package_name`
  - Web: `web:url`

### 3. Target Resolution

Resolves the target (group or OU) from:

- `group_key`: Looks up in groups_map
- `group_email`: Fetches via data source
- `ou_key`: Looks up in ou_map
- `ou_path`: Fetches via data source

### 4. Policy Generation

- Automatically creates the `InstallType` policy from the `install_type` field
- Adds any additional configurations from the `configurations` list
- Automatically adds `additional_target_keys` with the app_id (hidden from user!)

### 5. Resource Creation

Creates `googleworkspace_chrome_group_policy` or `googleworkspace_chrome_policy` resources with all policies and the app_id as an additional target key.

When migrating an extension from OU-based to group-based, the module ensures the group extension is created before the OU extension is removed. Both briefly coexist -- the Chrome Policy API resolves conflicts in favor of group policies.

## Validation

The module performs comprehensive validation at `terraform plan` time:

- Each extension has a valid app ID (`extension_id`, `android_app_id`, or `web_app_url`)
- Each extension has exactly one target (group OR OU, not both)
- All group and OU references are valid
- **No duplicate configurations**: The same extension cannot be configured multiple times for the same target, even across different files or between `extensions` and `extension_groups`

### Conflict Detection

If the same extension is configured for the same target in multiple places (with potentially different settings), terraform plan will fail:

```text
DUPLICATE EXTENSION CONFLICTS DETECTED:
Extension 'chrome:abc123' for group 'xxx' is configured 2 times:
  'my_extension' (in file1.yaml), 'group_ext_abc123' (from extension_group 'my_group' in file2.yaml).
Install types: [FORCED, BLOCKED]

Resolution: Remove duplicate configurations or consolidate into a single entry.
```

## Importing Existing Extensions

When adopting existing Chrome extension assignments that are already configured in Google Admin, add `import: true` to bring them into Terraform state without re-creating them.

**Requirements:**

- Provider `vdesouza/googleworkspace` with import support
- Terraform >= 1.7 (for `import` blocks with `for_each`)

**Workflow:**

1. Add `import: true` to each extension or extension_group entry in your YAML config:

  ```yaml
  extensions:
    ublock_origin_all:
      import: true
      extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
      ou_path: "/"
      install_type: "FORCED_AND_PIN_TO_TOOLBAR"

  extension_groups:
    developer_tools:
      import: true
      group_key: "engineering"
      install_type: "ALLOWED"
      extensions:
        - "bcjindcccaagfpapjjmafapmmgkkhgoa"
        - "fmkadmapgofadopljbjfkapdkoienihi"
  ```

2. Run `terraform plan` to verify the imports will succeed. The provider performs strict validation: it calls the Chrome Policy API to confirm each extension policy is **explicitly set** on the target OU/group (not just inherited from a parent).

3. Run `terraform apply` to import the resources into state.

4. Remove `import: true` from all YAML entries. This step is required -- Terraform errors if an `import` block targets a resource already in state.

5. Run `terraform plan` to confirm zero changes (clean state).

**Validation behavior:**

| Scenario | Result |
| --- | --- |
| Extension explicitly assigned to target | Import succeeds |
| Extension inherited from parent OU | Import fails with `"not explicitly set on... (inherited from...)"` |
| Invalid extension ID / app ID | Import fails with API error |
| Invalid target ID | Import fails with API error |

**Import ID format** (for manual `terraform import`):

Extension imports include the `app_id` as an additional target key:

```
<target_id>/app_id=<full_app_id>/<schema1>,<schema2>,...
```

Examples:

```bash
# OU-based Chrome extension
terraform import 'module.extensions.googleworkspace_chrome_policy.ou_extension["ublock_origin_all"]' \
  '03ph8a2z1shzb7m/app_id=chrome:cjpalhdlnbpafiamejdnhcphjbkeiagm/chrome.users.apps.InstallType'

# Group-based Chrome extension
terraform import 'module.extensions.googleworkspace_chrome_group_policy.group_extension["ublock_engineering"]' \
  '03ph8a2z1shzb7m/app_id=chrome:cjpalhdlnbpafiamejdnhcphjbkeiagm/chrome.users.apps.InstallType'

# Android app with configurations
terraform import 'module.extensions.googleworkspace_chrome_policy.ou_extension["google_keep"]' \
  '03ph8a2z1shzb7m/app_id=android:com.google.android.keep/chrome.users.apps.InstallType,chrome.users.apps.ManagedConfiguration'
```

## References

- [Chrome Policy API - Apps](https://developers.google.com/chrome/policy/guides/policy-schemas)
- [Chrome Web Store](https://chrome.google.com/webstore/category/extensions)
- [Managing Chrome Apps and Extensions](https://support.google.com/chrome/a/answer/2649489)
- [Terraform Google Workspace Provider](https://registry.terraform.io/providers/vdesouza/googleworkspace)
