output "group_policies" {
  description = "Map of all group-based policies created"
  value = {
    for key, policy in googleworkspace_chrome_group_policy.group_policy :
    key => {
      id       = policy.id
      group_id = policy.group_id
      type     = "group"
      policies = [
        for p in policy.policies : {
          schema_name   = p.schema_name
          schema_values = p.schema_values
        }
      ]
      additional_target_keys = policy.additional_target_keys
    }
  }
}

output "ou_policies" {
  description = "Map of all OU-based policies created"
  value = {
    for key, policy in googleworkspace_chrome_policy.ou_policy :
    key => {
      id          = policy.id
      org_unit_id = policy.org_unit_id
      type        = "ou"
      policies = [
        for p in policy.policies : {
          schema_name   = p.schema_name
          schema_values = p.schema_values
        }
      ]
      additional_target_keys = policy.additional_target_keys
    }
  }
}

output "all_policies" {
  description = "Map of all policies created (both group and OU-based)"
  value = merge(
    {
      for key, policy in googleworkspace_chrome_group_policy.group_policy :
      key => {
        id        = policy.id
        target_id = policy.group_id
        type      = "group"
        policies = [
          for p in policy.policies : {
            schema_name   = p.schema_name
            schema_values = p.schema_values
          }
        ]
      }
    },
    {
      for key, policy in googleworkspace_chrome_policy.ou_policy :
      key => {
        id        = policy.id
        target_id = policy.org_unit_id
        type      = "ou"
        policies = [
          for p in policy.policies : {
            schema_name   = p.schema_name
            schema_values = p.schema_values
          }
        ]
      }
    }
  )
}

output "policy_count" {
  description = "Total number of policies created (group + OU)"
  value = (
    length(googleworkspace_chrome_group_policy.group_policy) +
    length(googleworkspace_chrome_policy.ou_policy)
  )
}

output "group_policy_count" {
  description = "Number of group-based policies created"
  value       = length(googleworkspace_chrome_group_policy.group_policy)
}

output "ou_policy_count" {
  description = "Number of OU-based policies created"
  value       = length(googleworkspace_chrome_policy.ou_policy)
}

output "policies_by_group" {
  description = "Map of group IDs to their associated policies"
  value = {
    for group_id in distinct([for p in googleworkspace_chrome_group_policy.group_policy : p.group_id]) :
    group_id => {
      policy_keys = [
        for key, policy in googleworkspace_chrome_group_policy.group_policy :
        key if policy.group_id == group_id
      ]
      count = length([
        for policy in googleworkspace_chrome_group_policy.group_policy :
        policy if policy.group_id == group_id
      ])
    }
  }
}

output "policies_by_ou" {
  description = "Map of OU IDs to their associated policies"
  value = {
    for ou_id in distinct([for p in googleworkspace_chrome_policy.ou_policy : p.org_unit_id]) :
    ou_id => {
      policy_keys = [
        for key, policy in googleworkspace_chrome_policy.ou_policy :
        key if policy.org_unit_id == ou_id
      ]
      count = length([
        for policy in googleworkspace_chrome_policy.ou_policy :
        policy if policy.org_unit_id == ou_id
      ])
    }
  }
}

# Output for group_priority module - includes group_email for ordering lookup
output "group_policies_detail" {
  description = "Detailed map of group policies including group email for priority ordering"
  value = {
    for key, config in local.group_policies :
    key => {
      group_id = config.resolved_group_id
      group_email = coalesce(
        lookup(config, "group_email", null),
        lookup(config, "group_key", null) != null ? var.groups_map[config.group_key].email : null
      )
      policies = [
        for p in config.policies : {
          schema_name = p.schema_name
        }
      ]
    }
  }
}

# Outputs for policy_orderings defined in policy YAML files
output "policy_orderings" {
  description = "Map of policy-specific orderings defined in policy YAML files"
  value       = local.policy_orderings_from_yaml
}

output "policy_orderings_by_file" {
  description = "Map of policy orderings grouped by source file for validation"
  value       = local.policy_orderings_by_file
}

output "policy_orderings_with_source" {
  description = "Map of policy orderings with their source file information"
  value       = local.all_policy_orderings_with_source
}

# Import support outputs
output "ou_policies_to_import" {
  description = "Map of OU policies with import: true and their computed import IDs"
  value       = local.ou_policies_to_import
}

output "group_policies_to_import" {
  description = "Map of group policies with import: true and their computed import IDs"
  value       = local.group_policies_to_import
}
