terraform {
  required_version = ">= 1.3"

  required_providers {
    googleworkspace = {
      source  = "vdesouza/googleworkspace"
      version = ">= 1.3.13"
    }
  }
}

module "assets" {
  source = "../../../modules/assets"

  yaml_config_path   = "${path.module}/config/assets"
  assets_source_path = "${path.module}/assets"
}

output "assets_map" {
  value = module.assets.assets_map
}
