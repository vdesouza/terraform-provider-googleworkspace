# Assets Module

This Terraform module manages Chrome Policy file uploads from YAML configuration files. Files are uploaded directly to the Chrome Policy API and can be referenced in Chrome policies (e.g., wallpapers, avatars, terms of service).

## Overview

The module reads YAML configuration files that define assets to be uploaded. It uses the `googleworkspace_chrome_policy_file` resource to upload files to Google's Chrome Policy storage, which returns a `download_uri` that can be used in Chrome policies.

## Features

- Upload files directly to Chrome Policy API
- Automatic file change detection via SHA256 hash
- Support for multiple policy fields per asset
- Single file can be uploaded for multiple policy types
- YAML-based configuration for easy management
- Integration with the policies module via asset references

## Usage

### Basic Usage

```hcl
module "assets" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/assets?ref=v1.4.0"

  yaml_config_path   = "${path.module}/config/assets"
  assets_source_path = "${path.module}/config/assets"
}
```

### With Policies Module Integration

```hcl
module "assets" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/assets?ref=v1.4.0"

  yaml_config_path   = "${path.module}/config/assets"
  assets_source_path = "${path.module}/config/assets"
}

module "policies" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/policies?ref=v1.4.0"

  yaml_config_path = "${path.module}/config/policies"
  assets_map       = module.assets.assets_map
}
```

## YAML Configuration

Place your YAML files in the assets configuration directory. Each file should have an `assets` key containing asset configurations. See [YAML_SCHEMA.md](YAML_SCHEMA.md) for full schema details.

### Example Configuration

```yaml
assets:
  default_wallpaper:
    file: "wallpapers/defaultwallpaper.jpeg"
    description: "Default wallpaper for all managed devices"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
    comments: |
      This wallpaper is deployed to all managed ChromeOS devices.

  corporate_branding:
    file: "images/corporate.jpeg"
    description: "Corporate image for multiple uses"
    content_type: "image/jpeg"
    policy_fields:
      - "chrome.users.Wallpaper.wallpaperImage"
      - "chrome.users.Avatar.userAvatarImage"
```

### Asset Properties

| Property | Description | Required |
|----------|-------------|----------|
| file | Path to the asset file relative to `assets_source_path` | yes |
| policy_fields | List of Chrome Policy schema fields to upload for | yes |
| description | Human-readable description of the asset | no |
| content_type | MIME type of the file (e.g., `image/jpeg`) | no |
| comments | Additional comments or notes | no |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| yaml_config_path | Path to directory containing YAML asset configuration files | string | - | yes |
| assets_source_path | Path to directory containing the actual asset files | string | - | yes |

## Outputs

| Name | Description |
|------|-------------|
| assets_map | Map of all assets with metadata and download URIs (key: `asset_key/policy_field`) |
| assets_by_key | Map of assets grouped by asset key with all their policy field uploads |

### Output Structure

The `assets_map` output provides:

```hcl
{
  "default_wallpaper/chrome.users.Wallpaper.wallpaperImage" = {
    asset_key    = "default_wallpaper"
    policy_field = "chrome.users.Wallpaper.wallpaperImage"
    file         = "wallpapers/defaultwallpaper.jpeg"
    description  = "Default wallpaper for all managed devices"
    comments     = "..."
    content_type = "image/jpeg"
    download_uri = "https://chromepolicy.googleapis.com/..."  # Use this in policies
    file_hash    = "abc123..."  # SHA256 hash for change detection
  }
}
```

The `assets_by_key` output provides:

```hcl
{
  "default_wallpaper" = {
    file        = "wallpapers/defaultwallpaper.jpeg"
    description = "Default wallpaper for all managed devices"
    comments    = "..."
    policy_fields = {
      "chrome.users.Wallpaper.wallpaperImage" = "https://chromepolicy.googleapis.com/..."
    }
  }
}
```

## Referencing Assets in Policies

Assets can be referenced in policy YAML files using the syntax `${asset.<asset_key>/<policy_field>}`:

```yaml
policies:
  default_wallpaper_policy:
    group_email: "all-users@example.com"
    policies:
      - schema_name: "chrome.users.Wallpaper"
        schema_values:
          wallpaperImage:
            downloadUri: "${asset.default_wallpaper/chrome.users.Wallpaper.wallpaperImage}"
```

The policies module will automatically resolve this reference to the `download_uri` from the Chrome Policy API.

## Supported Policy Fields

### Wallpapers (`chrome.users.Wallpaper.wallpaperImage`)

- **Format**: JPEG only
- **Max Size**: 16384 KB (16 MB)
- **Usage**: Sets default wallpaper for ChromeOS devices

### Avatars (`chrome.users.Avatar.userAvatarImage`)

- **Format**: JPEG only
- **Max Size**: 512 KB
- **Usage**: Sets custom avatar for user accounts

### Terms of Service (`chrome.users.CustomTermsOfService.termsOfServiceUrl`)

- **Format**: Plain text
- **Max Size**: 512 KB
- **Usage**: Custom terms of service for public sessions

## Directory Structure

```text
config/assets/
├── example_assets.yaml       # Asset definitions
├── wallpapers/
│   ├── defaultwallpaper.jpeg
│   └── alternativewallpaper.jpeg
├── avatars/
│   └── corporate.jpeg
└── tos/
    └── terms.txt
```

## Requirements

- Terraform >= 1.3
- googleworkspace provider >= 1.3.13
- OAuth scope: `https://www.googleapis.com/auth/chrome.management.policy`

## Resources Created

- `googleworkspace_chrome_policy_file`: One resource per asset/policy_field combination

## Notes

- Files are uploaded directly to Google's Chrome Policy storage (not GCS)
- The `download_uri` returned by the API is specific to Chrome Policy
- File changes are detected automatically via SHA256 hash
- Re-uploading a file generates a new `download_uri`
- A single file can be uploaded for multiple policy fields

## References

- [Chrome Policy Media Upload API](https://developers.google.com/chrome/policy/reference/rest/v1/media/upload)
- [Chrome Policy Schema Reference](https://developers.google.com/chrome/policy/guides/policy-schemas)
- [Terraform Google Workspace Provider](https://registry.terraform.io/providers/vdesouza/googleworkspace)
