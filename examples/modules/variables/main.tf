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

module "policies" {
  source = "../../../modules/policies"

  yaml_config_paths = ["${path.module}/config/policies"]
  variables_map     = module.variables.variables_map
  assets_map        = {}
}

output "variables_map" {
  value = module.variables.variables_map
}

output "variable_names" {
  value = module.variables.variable_names
}
