terraform {
  required_version = ">= 1.3"

  required_providers {
    googleworkspace = {
      source  = "vdesouza/googleworkspace"
      version = ">= 1.3.13"
    }
  }
}

module "variables" {
  source = "../../../modules/variables"

  yaml_config_path = "${path.module}/config/variables"
}

module "groups" {
  source = "../../../modules/groups"

  yaml_config_path = "${path.module}/config/groups"
  variables_map    = module.variables.variables_map
}

output "all_groups" {
  value = module.groups.all_groups
}
