# Extensions YAML Configuration Schema Reference

Extensions can be defined in any YAML file that the extensions module reads - including policy YAML files. This allows you to keep related configurations together.

## Quick Reference

### Extension Group - Multiple Extensions with Shared Settings

```yaml
extension_groups:
  developer_tools:
    group_key: "developers"
    install_type: "ALLOWED"
    extensions:
      - "bcjindcccaagfpapjjmafapmmgkkhgoa"  # JSON Formatter
      - "fmkadmapgofadopljbjfkapdkoienihi"  # React DevTools
```

### Chrome Extension - Force Install

```yaml
extensions:
  ublock_origin:
    extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
    group_key: "all_users"
    install_type: "FORCED"
```

### Chrome Extension - Allow

```yaml
extensions:
  google_translate:
    extension_id: "aapbdbdomjkkjkaonfhkkikfgjllcleb"
    group_email: "employees@example.com"
    install_type: "ALLOWED"
```

### Android App

```yaml
extensions:
  google_keep:
    android_app_id: "com.google.android.keep"
    group_key: "chromeos_users"
    install_type: "FORCED"
```

### Web App

```yaml
extensions:
  google_docs:
    web_app_url: "https://docs.google.com"
    group_email: "all@example.com"
    install_type: "FORCED"
```

## Top-Level Structure

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `extensions` | map | No | Map of individual extension configurations keyed by a unique identifier |
| `extension_groups` | map | No | Map of extension groups for bulk configuration with shared settings |
| `extension_orderings` | map | No | Extension-specific group orderings (extension_id -> list of group emails) |

**Note**: At least one of `extensions` or `extension_groups` should be defined.

**Important**: `default_extension_ordering` must be defined in group_priority YAML files (`config/group_priority/*.yaml`), not in extension YAML files.

## Extension Group Ordering

When multiple groups have the same extension applied, group ordering determines which group's settings take precedence. Groups listed first have higher priority.

**Where orderings can be defined:**

- `default_extension_ordering`: **Only** in group priority YAML files (`config/group_priority/*.yaml`)
- `extension_orderings`: Either in extension YAML files OR group priority YAML files

Each extension can only have ONE ordering definition across all files.

### Default Extension Ordering

The default ordering applies to all extensions and must be defined in group priority YAML files:

```yaml
# config/group_priority/priority.yaml
# This is the ONLY place default_extension_ordering should be defined
default_extension_ordering:
  - "security-team@example.com"    # Highest priority
  - "engineering@example.com"
  - "all-users@example.com"        # Lowest priority
```

### Extension-Specific Orderings

Override the default ordering for specific extensions. These can be defined in extension YAML files:

```yaml
# config/extensions/my_extensions.yaml
extension_orderings:
  # Extension ID (without chrome:/android:/web: prefix)
  cjpalhdlnbpafiamejdnhcphjbkeiagm:  # uBlock Origin
    - "engineering@example.com"      # Higher priority for this extension
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

### Ordering Validation Rules

The extensions module parses `extension_orderings` and passes them to the `group_priority` module, which validates:

1. **Only one default**: `default_extension_ordering` can only be defined in group_priority YAML files
2. **Only one per extension**: Each extension can only have one ordering definition (across all sources)
3. **All groups must be included**: Every group that has an extension applied must be in the ordering (either default or extension-specific)
4. **Groups must exist**: All groups referenced in orderings must exist in `groups_map`

## Extension Groups

Extension groups allow you to configure multiple extensions with shared settings, reducing repetition.

### Extension Group Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `extensions` | list or string | Yes | List of extension IDs, or a variable reference (e.g., `"{var_name}"`) |
| `group_key` | string | One target required | Key referencing a group from the groups module (supports variables) |
| `group_email` | string | One target required | Direct email address of an existing Google Group (supports variables) |
| `ou_key` | string | One target required | Key referencing an OU from the ou_map (supports variables) |
| `ou_path` | string | One target required | Full path to an organizational unit (supports variables) |
| `install_type` | string | No | How extensions should be installed |
| `configurations` | list or string | No | Additional configurations, or a variable reference (e.g., `"{config_var}"`) |

### Variable Support in Extension Groups

The `extensions` field in `extension_groups` supports variable references. This allows reusable lists of extension IDs:

```yaml
# config/variables/extension_lists.yaml
developer_tools_list:
  - "bcjindcccaagfpapjjmafapmmgkkhgoa"  # JSON Formatter
  - "fmkadmapgofadopljbjfkapdkoienihi"  # React DevTools
```

```yaml
# config/extensions/example.yaml
extension_groups:
  # Use a variable for the entire list
  developer_tools:
    group_key: "engineering"
    install_type: "ALLOWED"
    extensions: "{developer_tools_list}"
```

You can also mix direct IDs with variable references in a list:

```yaml
extension_groups:
  mixed_list:
    group_key: "engineering"
    install_type: "ALLOWED"
    extensions:
      - "directextensionid123"      # Direct ID
      - "{developer_tools_list}"    # Variable (expands to multiple IDs)
      - "anotherextension456"       # Direct ID
```

### Configuration Variables

The `configurations` field also supports variable references for reusable configuration blocks:

```yaml
# config/variables/configs.yaml
blocked_dangerous_permissions:
  - blockedPermissions:
      - "webRequest"
      - "proxy"
      - "debugger"

standard_managed_config:
  - managedConfiguration: |
      {
        "apiEndpoint": "https://api.company.com",
        "enableTelemetry": true
      }
```

```yaml
# config/extensions/security.yaml
extension_groups:
  security_extensions:
    group_key: "all_users"
    install_type: "FORCED"
    configurations: "{blocked_dangerous_permissions}"  # Resolves to config list
    extensions:
      - "cjpalhdlnbpafiamejdnhcphjbkeiagm"
```

### Extension Group Examples

```yaml
extension_groups:
  # Allow developer tools for engineering team
  developer_tools:
    group_key: "engineering"
    install_type: "ALLOWED"
    extensions:
      - "bcjindcccaagfpapjjmafapmmgkkhgoa"  # JSON Formatter
      - "fmkadmapgofadopljbjfkapdkoienihi"  # React DevTools
      - "lmhkpmbekcpmknklioeibfkpmmfibljd"  # Redux DevTools

  # Block unwanted extensions for everyone
  blocked_list:
    group_key: "all_users"
    install_type: "BLOCKED"
    extensions:
      - "suspiciousextension123"
      - "cryptominerextension456"

  # Force install with shared configurations
  security_tools:
    group_key: "all_users"
    install_type: "FORCED"
    configurations:
      - blockedPermissions:
          - "webRequest"
    extensions:
      - "cjpalhdlnbpafiamejdnhcphjbkeiagm"  # uBlock Origin
      - "gcbommkclmclpchllfjekcdonpmejbdp"  # HTTPS Everywhere
```

### How Extension Groups Work

Each extension in the `extensions` list is expanded into an individual extension entry with:

- All shared settings from the group (install_type, target, configurations)
- A unique key generated as `{group_name}_{extension_id_prefix}`
- The extension_id set to the list item value

Individual `extensions` entries take precedence over expanded `extension_groups` entries if there are key conflicts.

## Extension Configuration Fields

### App Identification (One Required)

| Field | Type | Description | Auto-Prefix |
|-------|------|-------------|-------------|
| `extension_id` | string | Chrome Web Store extension ID | `chrome:` |
| `chrome_extension_id` | string | Same as `extension_id` | `chrome:` |
| `android_app_id` | string | Android package name | `android:` |
| `web_app_url` | string | Full URL for web apps | `web:` |
| `app_id` | string | Pre-formatted app_id with prefix | None (used as-is) |

**Note**: Only ONE of these fields should be specified per extension.

### Target (One Required)

| Field | Type | Description |
|-------|------|-------------|
| `group_key` | string | Key referencing a group from the groups module |
| `group_email` | string | Direct email address of an existing Google Group |
| `ou_key` | string | Key referencing an OU from the ou_map |
| `ou_path` | string | Full path to an organizational unit (e.g., `/Engineering`) |

**Note**: Only ONE of these fields should be specified per extension.

### Installation Settings

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `install_type` | string | No | None | How the extension should be installed |

#### Install Type Values

| Value | Description | Available For |
|-------|-------------|---------------|
| `FORCED` | Force install; users cannot remove | Chrome, Android, Web |
| `ALLOWED` | Users can choose to install | Chrome, Android, Web |
| `BLOCKED` | Prevent installation | Chrome, Android |
| `FORCED_AND_PIN_TO_TOOLBAR` | Force install and pin to toolbar | Chrome, Android, Web |
| `NORMAL` | Force install but users can disable | Chrome only |
| `NORMAL_AND_PIN_TO_TOOLBAR` | Force install, pin, users can disable | Chrome only |
| `REMOVE` | Block and remove from devices | Chrome only |

### Advanced Configuration (Optional)

| Field | Type | Description |
|-------|------|-------------|
| `configurations` | list | Additional extension-specific configurations |

#### Simplified Configuration Format (Recommended)

Use field names directly - the module automatically maps them to the correct Chrome policy schemas:

```yaml
configurations:
  - blockedPermissions:
      - "geolocation"
  - allowedPermissions:
      - "storage"
  - managedConfiguration: |
      {"apiKey": "value"}
```

#### Supported Simplified Fields

| Field Name | Auto-maps to Schema |
|------------|---------------------|
| `blockedPermissions` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `allowedPermissions` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `blockedHosts` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `allowedHosts` | `chrome.users.apps.PermissionsAndUrlAccess` |
| `managedConfiguration` | `chrome.users.apps.ManagedConfiguration` |
| `allowAccessToKeys` | `chrome.users.apps.CertificateManagement` |
| `includeInChromeWebStoreCollection` | `chrome.users.apps.IncludeInChromeWebStoreCollection` |
| `defaultLaunchContainer` | `chrome.users.apps.DefaultLaunchContainer` |
| `skipPrintConfirmation` | `chrome.users.apps.SkipPrintConfirmation` |
| `installationUrl` | `chrome.users.apps.InstallationUrl` |

**Note**: Fields that map to the same schema (e.g., `blockedPermissions` and `allowedPermissions`) are automatically merged.

#### Legacy Configuration Format

You can still use the explicit schema format if needed:

```yaml
configurations:
  - schema_name: "chrome.users.apps.PolicyName"
    schema_values:
      fieldName: value
```

## How App IDs Work

The Chrome Policy API requires app IDs to be prefixed with the app type. This module handles that automatically:

| You Provide | Module Generates |
|-------------|------------------|
| `extension_id: "abc123"` | `chrome:abc123` |
| `chrome_extension_id: "abc123"` | `chrome:abc123` |
| `android_app_id: "com.example.app"` | `android:com.example.app` |
| `web_app_url: "https://example.com"` | `web:https://example.com` |
| `app_id: "chrome:abc123"` | `chrome:abc123` (used as-is) |

## Complete Examples

### Security Extensions

```yaml
extensions:
  # Force install uBlock Origin for everyone
  ublock_origin_all:
    extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
    group_key: "all_users"
    install_type: "FORCED"

  # Force install and pin LastPass for security team
  lastpass_security:
    extension_id: "hdokiejnpimakedhajhdlcegeplioahd"
    group_email: "security-team@company.com"
    install_type: "FORCED_AND_PIN_TO_TOOLBAR"

  # Block a suspicious extension
  blocked_crypto_miner:
    extension_id: "suspiciousextensionid123"
    group_key: "all_users"
    install_type: "BLOCKED"
```

### Developer Tools

```yaml
extensions:
  # Allow JSON Formatter for developers
  json_formatter_devs:
    extension_id: "bcjindcccaagfpapjjmafapmmgkkhgoa"
    group_email: "developers@company.com"
    install_type: "ALLOWED"

  # Tampermonkey with specific permissions (simplified format)
  tampermonkey_devs:
    extension_id: "dhdgffkkebhmkfjojejmpbldmpobfkfo"
    group_key: "developers"
    install_type: "ALLOWED"
    configurations:
      - blockedPermissions:
          - "geolocation"
      - allowedPermissions:
          - "storage"
          - "tabs"

  # React DevTools - forced for frontend team
  react_devtools_frontend:
    extension_id: "fmkadmapgofadopljbjfkapdkoienihi"
    group_email: "frontend@company.com"
    install_type: "FORCED"
```

### Mobile Apps (ChromeOS)

```yaml
extensions:
  # Google Keep for all ChromeOS users
  google_keep_chromeos:
    android_app_id: "com.google.android.keep"
    ou_path: "/ChromeOS Users"
    install_type: "FORCED"

  # Slack for employees
  slack_android:
    android_app_id: "com.Slack"
    group_key: "all_employees"
    install_type: "ALLOWED"
```

### Web Apps

```yaml
extensions:
  # Google Docs as a web app
  google_docs_webapp:
    web_app_url: "https://docs.google.com"
    group_key: "all_users"
    install_type: "FORCED"

  # Internal tool as web app
  internal_dashboard:
    web_app_url: "https://dashboard.internal.company.com"
    group_email: "all-employees@company.com"
    install_type: "FORCED_AND_PIN_TO_TOOLBAR"
```

### OU-Based Deployment

```yaml
extensions:
  # Chrome Remote Desktop for support team (entire OU)
  chrome_remote_desktop_support:
    extension_id: "inomeogfingihgjfjlpeplalcfajhgai"
    ou_path: "/Support Team"
    install_type: "FORCED"

  # Kiosk-specific extension
  kiosk_extension:
    extension_id: "kioskextensionid123"
    ou_path: "/Kiosks/Reception"
    install_type: "FORCED"
```

### Advanced: Managed Configuration

```yaml
extensions:
  # Extension with JSON configuration (simplified format)
  managed_extension:
    extension_id: "managedextensionid456"
    group_key: "sales_team"
    install_type: "FORCED"
    configurations:
      - managedConfiguration: |
          {
            "apiEndpoint": "https://api.company.com",
            "features": {
              "featureA": true,
              "featureB": false
            },
            "maxItems": 100
          }
```

## Available Extension Configurations

These configurations can be used in the `configurations` field:

| Schema Name | Description |
|-------------|-------------|
| `chrome.users.apps.InstallType` | Installation mode (auto-generated from `install_type`) |
| `chrome.users.apps.PermissionsAndUrlAccess` | Control extension permissions |
| `chrome.users.apps.ManagedConfiguration` | Provide JSON configuration to extension |
| `chrome.users.apps.CertificateManagement` | Certificate access control |
| `chrome.users.apps.AccessToKeys` | Control access to client keys |
| `chrome.users.apps.EnterpriseChallenge` | Enterprise challenge key access |
| `chrome.users.apps.IncludeInChromeWebStoreCollection` | Web Store collection visibility |
| `chrome.users.apps.DefaultLaunchContainer` | Default launch mode |
| `chrome.users.apps.SkipPrintConfirmation` | Skip print dialogs |
| `chrome.users.apps.InstallationUrl` | Custom installation URL |

## Validation Rules

The module performs validation at `terraform plan` time and will fail with clear error messages if any rules are violated.

1. **One app identifier required**: Each extension must have exactly one of: `extension_id`, `chrome_extension_id`, `android_app_id`, `web_app_url`, or `app_id`

2. **One target required**: Each extension must target exactly one of: `group_key`, `group_email`, `ou_key`, or `ou_path`

3. **Valid references**: All `group_key` and `ou_key` references must exist in the respective maps

4. **No target conflicts**: Cannot specify both group and OU targets for the same extension

5. **No duplicate extension+target combinations**: The same extension cannot be configured multiple times for the same target (group or OU), even across different files or between `extensions` and `extension_groups`. This prevents conflicting configurations like:
   - Extension X set to BLOCKED in one file but ALLOWED in another
   - Same extension appearing in both `extensions` and `extension_groups` for the same target

### Example Conflict Error

If you have the same extension configured twice for the same group:

```yaml
# File 1: security.yaml
extensions:
  ublock_all:
    extension_id: "cjpalhdlnbpafiamejdnhcphjbkeiagm"
    group_key: "all_users"
    install_type: "FORCED"

# File 2: blocklist.yaml
extension_groups:
  blocked_list:
    group_key: "all_users"
    install_type: "BLOCKED"
    extensions:
      - "cjpalhdlnbpafiamejdnhcphjbkeiagm"  # Same extension!
```

Terraform plan will fail with:

```text
DUPLICATE EXTENSION CONFLICTS DETECTED:
Extension 'chrome:cjpalhdlnbpafiamejdnhcphjbkeiagm' for group 'xxx' is configured 2 times:
  'ublock_all' (in security.yaml), 'blocked_list_cjpalhdlnbpa' (from extension_group 'blocked_list' in blocklist.yaml).
Install types: [FORCED, BLOCKED]
```

## References

- [Chrome Policy API - Apps Schemas](https://developers.google.com/chrome/policy/guides/policy-schemas)
- [Managing Chrome Apps and Extensions](https://support.google.com/chrome/a/answer/2649489)
- [Force-install Apps Best Practices](https://support.google.com/chrome/a/answer/9962839)
