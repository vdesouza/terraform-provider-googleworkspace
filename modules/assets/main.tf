# Read all YAML files from the config directory
locals {
  # Get all YAML files in the directory
  yaml_files = fileset(var.yaml_config_path, "**/*.{yml,yaml}")

  # Parse all YAML files and merge them into a single map of assets
  # Each file should have an "assets" key containing asset configurations
  raw_assets = merge([
    for file in local.yaml_files : {
      for asset_key, asset_config in lookup(try(yamldecode(file("${var.yaml_config_path}/${file}")), {}), "assets", {}) :
      asset_key => merge(asset_config, {
        asset_key    = asset_key
        _source_file = file
      })
    }
  ]...)

  # Flatten assets to create one entry per policy_field
  # This allows a single file to be uploaded for multiple policy fields
  assets_by_policy_field = merge([
    for asset_key, asset in local.raw_assets : {
      for policy_field in asset.policy_fields :
      "${asset_key}/${policy_field}" => {
        asset_key    = asset_key
        file         = asset.file
        policy_field = policy_field
        description  = lookup(asset, "description", "")
        comments     = lookup(asset, "comments", "")
        content_type = lookup(asset, "content_type", "")
      }
    }
  ]...)

  # Also maintain a map by asset_key for easy reference
  all_assets = local.raw_assets
}

# Upload each asset to Chrome Policy API using googleworkspace_chrome_policy_file
# Creates one resource per policy_field, allowing the same file to be used for multiple policies
resource "googleworkspace_chrome_policy_file" "asset" {
  for_each = local.assets_by_policy_field

  file_path    = "${var.assets_source_path}/${each.value.file}"
  policy_field = each.value.policy_field
}
