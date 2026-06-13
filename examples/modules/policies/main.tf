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

module "policies" {
  source = "../../../modules/policies"

  yaml_config_paths = ["${path.module}/config/policies"]
  groups_map        = module.groups.all_groups
}

output "all_policies" {
  value = module.policies.all_policies
}
