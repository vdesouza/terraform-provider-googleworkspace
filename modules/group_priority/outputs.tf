output "policy_orderings" {
  description = "Map of policy schemas to their group priority orderings"
  value = {
    for schema, ordering in googleworkspace_chrome_policy_group_priority_ordering.ordering :
    schema => {
      policy_schema = ordering.policy_schema
      group_ids     = ordering.group_ids
    }
  }
}

output "policies_with_ordering" {
  description = "List of policy schemas that have group priority ordering configured"
  value       = keys(local.policies_needing_ordering)
}


output "policies_without_ordering" {
  description = "List of policy schemas that don't need ordering (single group or no groups)"
  value = [
    for schema in local.all_policy_schemas :
    schema
    if !contains(keys(local.policies_needing_ordering), schema)
  ]
}

output "default_policy_ordering" {
  description = "The default policy group ordering (by email)"
  value       = local.default_policy_ordering_emails
}

output "default_policy_ordering_source_file" {
  description = "The YAML file that defines the default policy ordering"
  value       = length(local.files_with_default_policy_ordering) == 1 ? local.files_with_default_policy_ordering[0] : null
}

output "policy_specific_orderings" {
  description = "Map of policy-specific orderings defined in YAML (merged from all sources)"
  value       = local.policy_orderings
}

output "local_policy_orderings" {
  description = "Map of policy orderings defined in group_priority YAML files"
  value       = local.local_policy_orderings
}

output "external_policy_orderings" {
  description = "Map of policy orderings from external sources (policy YAML files)"
  value       = var.external_policy_orderings
}

output "policy_orderings_with_source" {
  description = "Map of policy orderings with their source file information"
  value       = local.all_policy_orderings_with_source
}

output "duplicate_policy_orderings" {
  description = "Map of policy schemas that have duplicate ordering definitions (should be empty)"
  value       = local.duplicate_policy_orderings
}

# Extension ordering outputs
output "default_extension_ordering" {
  description = "The default extension group ordering (by email)"
  value       = local.default_extension_ordering_emails
}

output "default_extension_ordering_source_file" {
  description = "The YAML file that defines the default extension ordering"
  value       = length(local.files_with_default_extension_ordering) == 1 ? local.files_with_default_extension_ordering[0] : null
}

output "extension_orderings" {
  description = "Map of extension-specific orderings defined in group_priority YAML files"
  value       = local.local_extension_orderings
}

output "extension_orderings_with_source" {
  description = "Map of extension orderings with their source file information"
  value       = local.local_extension_orderings_with_source
}

output "validation_errors" {
  description = "List of validation errors (empty if all validations pass)"
  value       = local.validation_errors
}
