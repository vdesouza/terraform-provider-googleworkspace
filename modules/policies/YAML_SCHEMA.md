# YAML Configuration Schema Reference

## Quick Reference

### Group-Based Policy Configuration

```yaml
policies:
  policy_key:
    group_email: "group@example.com"
    policies:
      - schema_name: "chrome.users.WebKioskAccountRequiredToMakeChanges"
        schema_values:
          webKioskAccountRequiredToMakeChanges: true
```

### OU-Based Policy Configuration

```yaml
policies:
  policy_key:
    ou_path: "Engineering/Development"
    policies:
      - schema_name: "chrome.users.ChromeOsLockOnIdleSuspend"
        schema_values:
          chromeOsLockOnIdleSuspend: true
```

### Policy with Multiple Settings

```yaml
policies:
  policy_key:
    group_email: "group@example.com"
    policies:
      - schema_name: "chrome.users.PolicyName1"
        schema_values:
          setting1: "value1"
          setting2: 123
          setting3: true
      - schema_name: "chrome.users.PolicyName2"
        schema_values:
          anotherSetting: "value"
```

### Policy with Additional Target Keys

```yaml
policies:
  policy_key:
    ou_path: "Sales"
    policies:
      - schema_name: "chrome.users.AppInstallForceList"
        schema_values:
          appInstallForceList:
            - "extension_id1"
            - "extension_id2"
    additional_target_keys:
      - target_key: "app_id"
        target_value: "chrome:12345"
```

### Policy Referencing Group from Groups Module

```yaml
policies:
  policy_key:
    group_key: "chromepolicies_engineering_team"  # References a group defined in groups module
    policies:
      - schema_name: "chrome.users.ChromeOsLockOnIdleSuspend"
        schema_values:
          chromeOsLockOnIdleSuspend: true
```

### Policy Referencing OU by Key

```yaml
policies:
  policy_key:
    ou_key: "engineering_ou"  # References an OU from ou_map variable
    policies:
      - schema_name: "chrome.users.PasswordManagerEnabled"
        schema_values:
          passwordManagerEnabled: false
```

## Field Descriptions

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `policies` | map | ✓ | Top-level key containing all policy configurations |

### Policy Configuration Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `group_email` | string | ✓* | - | The email of the target group for this policy (group-based) |
| `group_key` | string | ✓* | - | Reference to a group defined in the groups module (alternative to group_email) |
| `ou_path` | string | ✓* | - | The organizational unit path for this policy (OU-based) |
| `ou_key` | string | ✓* | - | Reference to an OU from ou_map variable (alternative to ou_path) |
| `policies` | list | ✓ | - | List of policy schemas to apply (minimum 1) |
| `additional_target_keys` | list | | [] | Additional targeting parameters for fine-grained policy application |

*Note: Exactly one target must be specified:

- For group-based policies: Either `group_email` OR `group_key`
- For OU-based policies: Either `ou_path` OR `ou_key`
- Cannot specify both group and OU targets for the same policy

### Policies Block Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schema_name` | string | ✓ | The fully qualified name of the policy schema (e.g., "chrome.users.PolicyName") |
| `schema_values` | map | ✓ | Key-value pairs that correspond to the policy schema settings |

### Additional Target Keys Block Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target_key` | string | ✓ | The parameter name for additional targeting |
| `target_value` | string | ✓ | The parameter value for additional targeting |

## Schema Values Types

The `schema_values` field accepts various data types depending on the policy schema:

- **String**: `"value"`
- **Number**: `123`
- **Boolean**: `true` or `false`
- **List**: `["item1", "item2"]`
- **Map**: `{key: "value"}`

## Advanced Examples

### Multiple Policies for Engineering Group

```yaml
policies:
  engineering_security_policies:
    group_email: "engineering@example.com"
    policies:
      - schema_name: "chrome.users.ChromeOsLockOnIdleSuspend"
        schema_values:
          chromeOsLockOnIdleSuspend: true
      - schema_name: "chrome.users.PasswordManagerEnabled"
        schema_values:
          passwordManagerEnabled: false
      - schema_name: "chrome.users.IncognitoModeAvailability"
        schema_values:
          incognitoModeAvailability: 1  # Disabled
```

### Policy with Complex Schema Values

```yaml
policies:
  extension_policy:
    group_email: "developers@example.com"
    policies:
      - schema_name: "chrome.users.ExtensionSettings"
        schema_values:
          extensionSettings:
            "*":
              installation_mode: "blocked"
            "specific_extension_id":
              installation_mode: "force_installed"
              update_url: "https://clients2.google.com/service/update2/crx"
```

### Multiple Policy Configurations

```yaml
policies:
  contractor_policies:
    group_email: "contractors@example.com"
    policies:
      - schema_name: "chrome.users.DownloadRestrictions"
        schema_values:
          downloadRestrictions: 3  # Block dangerous downloads
      - schema_name: "chrome.users.AllowDeletingBrowserHistory"
        schema_values:
          allowDeletingBrowserHistory: false

  executive_policies:
    group_email: "executives@example.com"
    policies:
      - schema_name: "chrome.users.HomepageLocation"
        schema_values:
          homepageLocation: "https://company.com"
      - schema_name: "chrome.users.HomepageIsNewTabPage"
        schema_values:
          homepageIsNewTabPage: false
```

### Policy with Additional Targeting

```yaml
policies:
  app_specific_policy:
    group_email: "power-users@example.com"
    policies:
      - schema_name: "chrome.users.URLBlocklist"
        schema_values:
          urlBlocklist:
            - "example.com"
            - "*.badsite.com"
    additional_target_keys:
      - target_key: "app_id"
        target_value: "chrome:app_id_here"
      - target_key: "org_unit_id"
        target_value: "org_unit_id_here"
```

### OU-Based Security Policies

```yaml
policies:
  engineering_ou_security:
    ou_path: "Engineering/Development"
    policies:
      - schema_name: "chrome.users.PasswordManagerEnabled"
        schema_values:
          passwordManagerEnabled: false
      - schema_name: "chrome.users.ChromeOsLockOnIdleSuspend"
        schema_values:
          chromeOsLockOnIdleSuspend: true

  sales_ou_restrictions:
    ou_path: "Sales"
    policies:
      - schema_name: "chrome.users.IncognitoModeAvailability"
        schema_values:
          incognitoModeAvailability: 1
      - schema_name: "chrome.users.DownloadRestrictions"
        schema_values:
          downloadRestrictions: 2
```

### Mixed Group and OU Policies

```yaml
policies:
  # Group-based policy for specific team
  team_policy:
    group_email: "security-team@example.com"
    policies:
      - schema_name: "chrome.users.ExtensionInstallBlocklist"
        schema_values:
          extensionInstallBlocklist:
            - "*"

  # OU-based policy for entire department
  department_policy:
    ou_path: "Marketing"
    policies:
      - schema_name: "chrome.users.HomepageLocation"
        schema_values:
          homepageLocation: "https://marketing.example.com"
```

## Common Chrome Policy Schemas

Here are some commonly used Chrome policy schema names:

### Security Policies
- `chrome.users.PasswordManagerEnabled`
- `chrome.users.IncognitoModeAvailability`
- `chrome.users.ChromeOsLockOnIdleSuspend`
- `chrome.users.ScreenLockDelays`

### Browser Policies
- `chrome.users.HomepageLocation`
- `chrome.users.HomepageIsNewTabPage`
- `chrome.users.BookmarkBarEnabled`
- `chrome.users.DefaultSearchProviderEnabled`

### Extension Policies
- `chrome.users.ExtensionSettings`
- `chrome.users.ExtensionInstallBlocklist`
- `chrome.users.ExtensionInstallForcelist`

### Network Policies
- `chrome.users.URLBlocklist`
- `chrome.users.URLAllowlist`
- `chrome.users.ProxySettings`

### Download Policies
- `chrome.users.DownloadRestrictions`
- `chrome.users.DownloadDirectory`
- `chrome.users.AllowDeletingBrowserHistory`

## Target Reference Methods

### Group-Based Policies

You can reference groups in two ways:

#### 1. Direct Email Reference

```yaml
policies:
  policy_key:
    group_email: "engineering@example.com"
    policies: [...]
```

Use this when:
- The group is created outside Terraform
- You want to reference an existing group by email

#### 2. Group Module Reference

```yaml
policies:
  policy_key:
    group_key: "engineering_team"  # Must match a key in groups module
    policies: [...]
```

Use this when:
- The group is defined in the groups module
- You want Terraform to manage dependencies between groups and policies

### OU-Based Policies

You can reference organizational units in two ways:

#### 1. Direct Path Reference

```yaml
policies:
  policy_key:
    ou_path: "Engineering/Development"
    policies: [...]
```

Use this when:
- The OU path is known and static
- The OU is managed outside Terraform or already exists

#### 2. OU Map Reference

```yaml
policies:
  policy_key:
    ou_key: "engineering_dev_ou"  # Must match a key in ou_map variable
    policies: [...]
```

Use this when:
- You want to manage OUs in a separate module/configuration
- You want Terraform to manage dependencies between OUs and policies

## Asset References

Policies that require file uploads (like wallpapers, avatars, or terms of service) can reference assets from the assets module using the syntax: `${asset.<asset_key>/<policy_field>}`

### Wallpaper Policy with Asset Reference

```yaml
policies:
  corporate_wallpaper:
    group_email: "all-users@example.com"
    policies:
      - schema_name: "chrome.users.Wallpaper"
        schema_values:
          wallpaperImage:
            downloadUri: "${asset.default_wallpaper/chrome.users.Wallpaper.wallpaperImage}"
```

### Avatar Policy with Asset Reference

```yaml
policies:
  corporate_avatar:
    group_email: "all-users@example.com"
    policies:
      - schema_name: "chrome.users.Avatar"
        schema_values:
          userAvatarImage:
            downloadUri: "${asset.corporate_avatar/chrome.users.Avatar.userAvatarImage}"
```

The asset reference is automatically resolved to the `download_uri` returned by the Chrome Policy API when the asset was uploaded.

## Tips

1. **Schema Names**: Always use the fully qualified schema name (e.g., `chrome.users.PolicyName`)
2. **Schema Values**: Ensure your schema values match the expected format for each policy
3. **Testing**: Test policies with a small group before rolling out to larger groups
4. **Documentation**: Document why each policy is applied for future reference
5. **Group Dependencies**: When using `group_key`, ensure the groups module is deployed first
6. **Asset References**: Use the format `${asset.<asset_key>/<policy_field>}` to reference uploaded assets

## References

- [Chrome Policy List](https://chromeenterprise.google/policies/)
- [Chrome Management API](https://developers.google.com/chrome/management)
- [Policy Schemas Documentation](https://cloud.google.com/chrome-enterprise/docs/policies)
