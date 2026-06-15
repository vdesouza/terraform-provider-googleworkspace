terraform {
  required_version = ">= 1.3"

  required_providers {
    googleworkspace = {
      source  = "vdesouza/googleworkspace"
      version = ">= 1.3.13"
    }
  }
}

module "groups" {
  source = "../../../modules/groups"

  yaml_config_path = "${path.module}/config/groups"
}

module "extensions" {
  source = "../../../modules/extensions"

  yaml_config_paths = ["${path.module}/config/extensions"]
  groups_map        = module.groups.all_groups
}

output "all_extensions" {
  value = module.extensions.all_extensions
}
