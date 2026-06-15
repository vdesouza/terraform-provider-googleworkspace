output "group_extensions" {
  description = "Map of all group-based extension policies created"
  value = {
    for key, extension in googleworkspace_chrome_group_policy.group_extension :
    key => {
      id                     = extension.id
      group_id               = extension.group_id
      app_id                 = extension.additional_target_keys[0].target_value
      type                   = "group"
      policies               = extension.policies
      additional_target_keys = extension.additional_target_keys
    }
  }
}

output "ou_extensions" {
  description = "Map of all OU-based extension policies created"
  value = {
    for key, extension in googleworkspace_chrome_policy.ou_extension :
    key => {
      id                     = extension.id
      org_unit_id            = extension.org_unit_id
      app_id                 = extension.additional_target_keys[0].target_value
      type                   = "ou"
      policies               = extension.policies
      additional_target_keys = extension.additional_target_keys
    }
  }
}

output "all_extensions" {
  description = "Map of all extension policies created (both group and OU-based)"
  value = merge(
    {
      for key, extension in googleworkspace_chrome_group_policy.group_extension :
      key => {
        id        = extension.id
        target_id = extension.group_id
        app_id    = extension.additional_target_keys[0].target_value
        type      = "group"
        policies  = extension.policies
      }
    },
    {
      for key, extension in googleworkspace_chrome_policy.ou_extension :
      key => {
        id        = extension.id
        target_id = extension.org_unit_id
        app_id    = extension.additional_target_keys[0].target_value
        type      = "ou"
        policies  = extension.policies
      }
    }
  )
}

output "extension_count" {
  description = "Total number of extension policies created (group + OU)"
  value = (
    length(googleworkspace_chrome_group_policy.group_extension) +
    length(googleworkspace_chrome_policy.ou_extension)
  )
}

output "group_extension_count" {
  description = "Number of group-based extension policies created"
  value       = length(googleworkspace_chrome_group_policy.group_extension)
}

output "ou_extension_count" {
  description = "Number of OU-based extension policies created"
  value       = length(googleworkspace_chrome_policy.ou_extension)
}

output "extensions_by_group" {
  description = "Map of group IDs to their associated extension policies"
  value = {
    for group_id in distinct([for e in googleworkspace_chrome_group_policy.group_extension : e.group_id]) :
    group_id => {
      extension_keys = [
        for key, extension in googleworkspace_chrome_group_policy.group_extension :
        key if extension.group_id == group_id
      ]
      count = length([
        for extension in googleworkspace_chrome_group_policy.group_extension :
        extension if extension.group_id == group_id
      ])
    }
  }
}

output "extensions_by_ou" {
  description = "Map of OU IDs to their associated extension policies"
  value = {
    for ou_id in distinct([for e in googleworkspace_chrome_policy.ou_extension : e.org_unit_id]) :
    ou_id => {
      extension_keys = [
        for key, extension in googleworkspace_chrome_policy.ou_extension :
        key if extension.org_unit_id == ou_id
      ]
      count = length([
        for extension in googleworkspace_chrome_policy.ou_extension :
        extension if extension.org_unit_id == ou_id
      ])
    }
  }
}

output "validation_errors" {
  description = "List of validation errors detected (empty if no errors)"
  value = {
    duplicate_extension_conflicts = local.duplicate_extension_errors
    invalid_group_references      = local.invalid_group_extensions
    invalid_ou_references         = local.invalid_ou_extensions
    conflicting_targets           = local.conflicting_target_extensions
    missing_targets               = local.missing_target_extensions
    missing_app_ids               = local.missing_app_id_extensions
  }
}

# Output for group_priority module - includes group_email for ordering lookup
output "group_extensions_detail" {
  description = "Detailed map of group extensions including group email for priority ordering"
  value = {
    for key, config in local.group_extensions :
    key => {
      group_id = config.resolved_group_id
      group_email = coalesce(
        lookup(config, "group_email", null),
        lookup(config, "group_key", null) != null ? var.groups_map[config.group_key].email : null
      )
      app_id   = config.full_app_id
      policies = config.all_policies
    }
  }
}

# Outputs for extension_orderings defined in extension YAML files
output "extension_orderings" {
  description = "Map of extension-specific orderings defined in extension YAML files"
  value = merge([
    for file, orderings in local.local_extension_orderings_by_file :
    orderings
  ]...)
}

output "extension_orderings_by_file" {
  description = "Map of extension orderings grouped by source file for validation"
  value       = local.local_extension_orderings_by_file
}

output "extension_orderings_with_source" {
  description = "Map of extension orderings with their source file information"
  value       = local.local_extension_orderings_with_source
}

# Import support outputs
output "ou_extensions_to_import" {
  description = "Map of OU extensions with import: true and their computed import IDs"
  value       = local.ou_extensions_to_import
}

output "group_extensions_to_import" {
  description = "Map of group extensions with import: true and their computed import IDs"
  value       = local.group_extensions_to_import
}