# Primary output: Map of asset_key/policy_field to download URIs
# Use this to reference assets in Chrome policies
# Key format: "<asset_key>/<policy_field>"
# Example: "default_wallpaper/chrome.users.Wallpaper.wallpaperImage"
output "assets_map" {
  description = "Map of all assets with their metadata and download URIs for use in Chrome policies. Key format: asset_key/policy_field"
  value = {
    for key, asset in local.assets_by_policy_field :
    key => {
      asset_key    = asset.asset_key
      policy_field = asset.policy_field
      file         = asset.file
      description  = asset.description
      comments     = asset.comments
      content_type = asset.content_type
      # Download URI from Chrome Policy API - use this in Chrome policies
      download_uri = googleworkspace_chrome_policy_file.asset[key].download_uri
      # File hash for change detection
      file_hash = googleworkspace_chrome_policy_file.asset[key].file_hash
    }
  }
}

# Output organized by asset key, with all policy_fields and their download URIs
# Useful for seeing all uploads for a single asset file
output "assets_by_key" {
  description = "Map of assets grouped by asset key with all their policy field uploads"
  value = {
    for asset_key in distinct([for k, a in local.assets_by_policy_field : a.asset_key]) :
    asset_key => {
      file        = local.all_assets[asset_key].file
      description = lookup(local.all_assets[asset_key], "description", "")
      comments    = lookup(local.all_assets[asset_key], "comments", "")
      # Map of policy_field to download_uri for this asset
      policy_fields = {
        for key, asset in local.assets_by_policy_field :
        asset.policy_field => googleworkspace_chrome_policy_file.asset[key].download_uri
        if asset.asset_key == asset_key
      }
    }
  }
}
