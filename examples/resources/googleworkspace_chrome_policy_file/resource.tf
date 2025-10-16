# Example: Upload a wallpaper image for Chrome Policy
resource "googleworkspace_chrome_policy_file" "wallpaper" {
  file_path    = "${path.module}/wallpapers/company-wallpaper.jpg"
  policy_field = "chrome.users.WallpaperImage.value"
}

# Example: Use the uploaded file in a Chrome wallpaper policy
resource "googleworkspace_chrome_policy" "wallpaper_policy" {
  org_unit_id = "03ph8a2z2p7p8q3"

  policies {
    schema_name = "chrome.users.WallpaperImage"
    schema_values = {
      wallpaperImage = jsonencode({
        value = {
          downloadUri = googleworkspace_chrome_policy_file.wallpaper.download_uri
        }
      })
    }
  }
}

# Example: Upload multiple files
resource "googleworkspace_chrome_policy_file" "logo" {
  file_path    = "${path.module}/images/company-logo.png"
  policy_field = "chrome.users.WallpaperImage.value"
}

resource "googleworkspace_chrome_policy_file" "background" {
  file_path    = "${path.module}/images/login-background.jpg"
  policy_field = "chrome.users.WallpaperImage.value"
}

# Example: Upload with local variable
locals {
  wallpaper_path = "/path/to/your/wallpaper.png"
}

resource "googleworkspace_chrome_policy_file" "custom_wallpaper" {
  file_path    = local.wallpaper_path
  policy_field = "chrome.users.WallpaperImage.value"
}

output "wallpaper_uri" {
  description = "The download URI of the uploaded wallpaper"
  value       = googleworkspace_chrome_policy_file.wallpaper.download_uri
}
