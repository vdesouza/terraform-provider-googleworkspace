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

module "group_priority" {
  source = "../../../modules/group_priority"

  yaml_config_path = "${path.module}/config/group_priority"
  groups_map       = module.groups.all_groups
  policies_map     = module.policies.group_policies_detail

  external_policy_orderings             = module.policies.policy_orderings
  external_policy_orderings_with_source = module.policies.policy_orderings_with_source
}

output "policy_orderings" {
  value = module.group_priority.policy_orderings
}
