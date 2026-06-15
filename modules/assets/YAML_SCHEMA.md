# YAML Configuration Schema Reference

## Quick Reference

### Basic Asset Configuration

```yaml
assets:
  asset_key:
    file: "path/to/file.jpeg"
    description: "Description of the asset"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
```

### Asset with Multiple Policy Fields

```yaml
assets:
  corporate_image:
    file: "images/corporate.jpeg"
    description: "Corporate image used for multiple policies"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
      - "chrome.users.Avatar.userAvatarImage"
    comments: |
      This image is uploaded once but registered for multiple policy types.
```

### Multiple Assets

```yaml
assets:
  default_wallpaper:
    file: "wallpapers/default.jpeg"
    description: "Default wallpaper"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"

  engineering_wallpaper:
    file: "wallpapers/engineering.jpeg"
    description: "Engineering team wallpaper"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"

  profile_avatar:
    file: "avatars/profile.jpeg"
    description: "Profile avatar"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Avatar.userAvatarImage"
```

## Field Descriptions

### Top-Level Structure

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `assets` | map | Yes | Top-level key containing all asset configurations |

### Asset Configuration Fields

Each asset is defined under the `assets` key with a unique asset key.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `file` | string | Yes | - | Path to the asset file relative to `assets_source_path` |
| `policy_fields` | list | Yes | - | List of Chrome Policy schema fields to upload this file for |
| `description` | string | No | "" | Human-readable description of the asset |
| `content_type` | string | No | "" | MIME type of the file (e.g., `image/jpeg`, `text/plain`) |
| `comments` | string | No | "" | Additional comments or notes about the asset |

### Policy Fields

The `policy_fields` list specifies which Chrome Policy schema fields this file will be uploaded for. Each entry creates a separate upload to the Chrome Policy API.

Common policy fields:

| Policy Field                            | Description               |
|-----------------------------------------|---------------------------|
| `chrome.users.Wallpaper.wallpaperImage` | ChromeOS device wallpaper |

## Asset Key Format

Assets are referenced in policies using the format: `${asset.<asset_key>/<policy_field>}`

Example:

- Asset key: `default_wallpaper`
- Policy field: `chrome.users.Wallpaper.wallpaperImage`
- Full reference: `${asset.default_wallpaper/chrome.users.Wallpaper.wallpaperImage}`

## Supported Policy Fields

### Wallpapers (`chrome.users.Wallpaper.wallpaperImage`)

Used for ChromeOS device wallpaper policy.

**Constraints:**

- Format: JPEG only
- Max Size: 16384 KB (16 MB)
- Recommended: High resolution for various screen sizes

```yaml
assets:
  corporate_wallpaper:
    file: "wallpapers/corporate.jpeg"
    description: "Corporate wallpaper"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
```

### Avatars (`chrome.users.Avatar.userAvatarImage`)

Used for user avatar/profile picture policy.

**Constraints:**

- Format: JPEG only
- Max Size: 512 KB
- Recommended: Square aspect ratio

```yaml
assets:
  default_avatar:
    file: "avatars/default.jpeg"
    description: "Default user avatar"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Avatar.userAvatarImage"
```

### Terms of Service (`chrome.users.CustomTermsOfService.termsOfServiceUrl`)

Used for custom terms of service in public sessions.

**Constraints:**

- Format: Plain text only
- Max Size: 512 KB

```yaml
assets:
  tos:
    file: "tos/terms.txt"
    description: "Terms of service"
    content_type: "text/plain"
    policy_fields:
      - "chrome.users.CustomTermsOfService.termsOfServiceUrl"
```

## Referencing Assets in Policies

Assets are referenced in policy YAML files using the syntax: `${asset.<asset_key>/<policy_field>}`

### Wallpaper Policy Example

```yaml
# In config/assets/assets.yaml
assets:
  corporate_wallpaper:
    file: "wallpapers/corporate.jpeg"
    description: "Corporate wallpaper"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
```

```yaml
# In config/policies/policies.yaml
policies:
  corporate_wallpaper_policy:
    group_email: "all-users@example.com"
    policies:
      - schema_name: "chrome.users.Wallpaper"
        schema_values:
          wallpaperImage:
            downloadUri: "${asset.corporate_wallpaper/chrome.users.Wallpaper.wallpaperImage}"
```

### Avatar Policy Example

```yaml
# In config/assets/assets.yaml
assets:
  profile_avatar:
    file: "avatars/profile.jpeg"
    description: "Profile avatar"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Avatar.userAvatarImage"
```

```yaml
# In config/policies/policies.yaml
policies:
  profile_avatar_policy:
    ou_path: "Profiles"
    policies:
      - schema_name: "chrome.users.Avatar"
        schema_values:
          userAvatarImage:
            downloadUri: "${asset.profile_avatar/chrome.users.Avatar.userAvatarImage}"
```

### Multi-Purpose Asset Example

A single image file can be used for multiple policy types:

```yaml
# In config/assets/assets.yaml
assets:
  corporate_branding:
    file: "images/corporate_logo.jpeg"
    description: "Corporate branding image for wallpaper and avatar"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
      - "chrome.users.Avatar.userAvatarImage"
```

```yaml
# In config/policies/policies.yaml
policies:
  corporate_wallpaper:
    group_email: "all-users@example.com"
    policies:
      - schema_name: "chrome.users.Wallpaper"
        schema_values:
          wallpaperImage:
            downloadUri: "${asset.corporate_branding/chrome.users.Wallpaper.wallpaperImage}"

  corporate_avatar:
    group_email: "all-users@example.com"
    policies:
      - schema_name: "chrome.users.Avatar"
        schema_values:
          userAvatarImage:
            downloadUri: "${asset.corporate_branding/chrome.users.Avatar.userAvatarImage}"
```

## Advanced Examples

### Complete Asset Configuration

```yaml
assets:
  # Wallpapers
  corporate_default:
    file: "wallpapers/corporate_default.jpeg"
    description: "Default corporate wallpaper"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
    comments: |
      Primary wallpaper for all corporate devices.
      Features company logo and brand colors.

  partner_wallpaper:
    file: "wallpapers/partner.jpeg"
    description: "Partner/contractor wallpaper"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
    comments: |
      Distinct wallpaper for partner devices.

  # Avatars
  corporate_avatar:
    file: "avatars/corporate.jpeg"
    description: "Corporate user avatar"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Avatar.userAvatarImage"

  profile_avatar:
    file: "avatars/profile.jpeg"
    description: "Profile avatar"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Avatar.userAvatarImage"

  # Terms of service
  tos:
    file: "tos/terms.txt"
    description: "Terms of service"
    content_type: "text/plain"
    policy_fields:
      - "chrome.users.CustomTermsOfService.termsOfServiceUrl"
    comments: |
      Legal terms to be displayed.
```

## Directory Structure

Organize your assets directory to match your asset types:

```text
config/assets/
├── example_assets.yaml      # Asset definitions
├── wallpapers/
│   ├── corporate_default.jpeg
│   ├── engineering.jpeg
│   └── partner.jpeg
├── avatars/
│   ├── corporate.jpeg
│   └── profile.jpeg
└── tos/
    ├── terms.txt
    └── public_terms.txt
```

## Multiple YAML Files

You can split asset configurations across multiple YAML files:

```text
config/assets/
├── wallpapers.yaml          # Wallpaper asset definitions
├── avatars.yaml             # Avatar asset definitions
├── terms_of_service.yaml    # ToS asset definitions
├── wallpapers/
│   └── ...
├── avatars/
│   └── ...
└── tos/
    └── ...
```

All YAML files in the configuration directory will be read and merged.

## Tips

1. **Asset Keys**: Use descriptive, lowercase keys with underscores (e.g., `corporate_wallpaper`)
2. **File Organization**: Keep asset files in subdirectories for clarity
3. **Documentation**: Use `description` and `comments` fields to document asset purpose
4. **Content Type**: Always specify `content_type` for clarity
5. **Multi-Purpose Assets**: Use multiple `policy_fields` when the same file works for different policies
6. **File Size**: Keep files under the maximum size limits to avoid upload failures

## Validation

The module validates:

- File existence at the specified path
- Automatic change detection via SHA256 hash
- Terraform will re-upload files when content changes

## References

- [Chrome Policy File Upload API](https://developers.google.com/chrome/policy/reference/rest/v1/media/upload)
- [chrome.users.Wallpaper Schema](https://developers.google.com/chrome/policy/guides/policy-schemas)
- [chrome.users.Avatar Schema](https://developers.google.com/chrome/policy/guides/policy-schemas)
- [chrome.users.CustomTermsOfService Schema](https://developers.google.com/chrome/policy/guides/policy-schemas)
