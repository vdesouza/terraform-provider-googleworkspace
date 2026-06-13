# Read all YAML files from all config directories
locals {
  # Get all YAML files from all config directories
  yaml_files_by_path = {
    for path in var.yaml_config_paths :
    path => fileset(path, "**/*.{yml,yaml}")
  }

  # Flatten into a list of {path, file} objects
  all_yaml_files = flatten([
    for path, files in local.yaml_files_by_path : [
      for file in files : {
        path = path
        file = file
        key  = "${path}/${file}"
      }
    ]
  ])

  # Parse each YAML file individually to extract policies and policy_orderings
  parsed_yaml_files = {
    for item in local.all_yaml_files :
    item.key => try(yamldecode(file("${item.path}/${item.file}")), {})
  }

  # Parse all YAML files and merge them into a single map
  raw_policies = merge([
    for file, content in local.parsed_yaml_files : {
      for policy_key, policy_config in lookup(content, "policies", {}) :
      policy_key => merge(policy_config, {
        _source_file = file
      })
    }
  ]...)

  # ============================================================================
  # VARIABLE RESOLUTION
  # ============================================================================
  # Resolve {varname} references in policy configurations.
  # Variables are defined in config/variables/ and passed via variables_map.

  # Build lookup map for variable references
  # Key: "{varname}", Value: variable value (string or list)
  variable_lookup = {
    for var_name, var_value in var.variables_map :
    "{${var_name}}" => var_value
  }

  # Helper function to resolve a single value
  # If value exactly matches "{varname}", return the variable value (preserves type)
  # Otherwise return the original value
  resolve_variable_value = {
    for var_ref, var_value in local.variable_lookup :
    var_ref => var_value
  }

  # Resolve variables in policy configurations
  # Handles group_email, group_key, ou_path, ou_key fields
  policies_with_resolved_variables = {
    for policy_key, config in local.raw_policies :
    policy_key => merge(config, {
      # Resolve group_email if it's a variable reference
      group_email = lookup(config, "group_email", null) != null ? (
        lookup(local.variable_lookup, config.group_email, config.group_email)
      ) : null

      # Resolve group_key if it's a variable reference
      group_key = lookup(config, "group_key", null) != null ? (
        lookup(local.variable_lookup, config.group_key, config.group_key)
      ) : null

      # Resolve ou_path if it's a variable reference
      ou_path = lookup(config, "ou_path", null) != null ? (
        lookup(local.variable_lookup, config.ou_path, config.ou_path)
      ) : null

      # Resolve ou_key if it's a variable reference
      ou_key = lookup(config, "ou_key", null) != null ? (
        lookup(local.variable_lookup, config.ou_key, config.ou_key)
      ) : null
    })
  }

  # Extract policy_orderings from each YAML file
  # Maps: file -> { policy_schema -> [group_emails] }
  policy_orderings_by_file = {
    for file, content in local.parsed_yaml_files :
    file => lookup(content, "policy_orderings", {})
    if lookup(content, "policy_orderings", null) != null
  }

  # Flatten to get all policy schemas that have orderings defined
  # Maps: policy_schema -> { file, ordering }
  all_policy_orderings_with_source = merge([
    for file, orderings in local.policy_orderings_by_file : {
      for schema, ordering in orderings :
      schema => {
        source_file = file
        ordering    = ordering
      }
    }
  ]...)

  # Final merged policy_orderings (just schema -> ordering list)
  # Resolve variable references in orderings (e.g., "{admin_emails}" -> list of emails)
  policy_orderings_from_yaml = {
    for schema, config in local.all_policy_orderings_with_source :
    schema => (
      # Check if the ordering is a single variable reference (string that resolves to list)
      try(tostring(config.ordering), null) != null ? (
        lookup(local.variable_lookup, config.ordering, [config.ordering])
        ) : (
        # Otherwise it's already a list, resolve any variable references within
        flatten([
          for item in config.ordering :
          try(tostring(item), null) != null ? (
            # If item is a variable reference that resolves to a list, flatten it
            try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
          ) : [item]
        ])
      )
    )
  }

  # Helper to resolve asset references in string values
  # Asset references use format: ${asset.<asset_key>/<policy_field>}
  # Example: ${asset.default_wallpaper/chrome.users.Wallpaper.wallpaperImage}
  resolve_asset_value = {
    for asset_key, asset in var.assets_map :
    "$${asset.${asset_key}}" => asset.download_uri
  }

  # Function to resolve asset references in nested schema values
  # Converts schema_values with asset references to resolved URLs
  resolve_schema_values = {
    for policy_key, config in local.policies_with_resolved_variables :
    policy_key => [
      for policy in config.policies : {
        schema_name = policy.schema_name
        schema_values = {
          for k, v in policy.schema_values :
          k => (
            # If v is a string, check for asset reference
            try(tostring(v), null) != null ? (
              lookup(local.resolve_asset_value, tostring(v), v)
            ) :
            # For non-string values, use try() to handle lists vs maps
            # try() doesn't require consistent types between arguments
            try(
              # First: if v is a list/tuple, resolve asset refs preserving list type
              [for item in tolist(v) :
                try(lookup(local.resolve_asset_value, tostring(item), item), item)
              ],
              # Second: if v is a map, resolve asset refs in values
              { for nested_k, nested_v in v :
                nested_k => try(lookup(local.resolve_asset_value, tostring(nested_v), nested_v), nested_v)
              },
              # Fallback: return as-is
              v
            )
          )
        }
      }
    ]
  }

  # Extract all unique OU paths referenced in the configs (after variable resolution)
  all_ou_paths = distinct(compact([
    for key, config in local.policies_with_resolved_variables :
    lookup(config, "ou_path", null)
  ]))

  # Extract all unique group emails that are directly referenced (not via group_key)
  # (after variable resolution)
  all_group_emails = distinct(compact([
    for key, config in local.policies_with_resolved_variables :
    lookup(config, "group_email", null)
  ]))
}

# Fetch org unit data for all org units referenced in the configs
data "googleworkspace_org_unit" "ou_lookup" {
  for_each      = toset(local.all_ou_paths)
  org_unit_path = each.value
}

# Fetch group data for all groups referenced by email in the configs
data "googleworkspace_group" "group_lookup" {
  for_each = toset(local.all_group_emails)
  email    = each.value
}

locals {
  # Resolve target IDs from either group or OU references
  # Uses policies_with_resolved_variables which has {varname} references resolved
  policies_with_resolved_targets = {
    for key, config in local.policies_with_resolved_variables :
    key => merge(config, {
      # Determine policy type (group or ou)
      is_group_policy = (
        lookup(config, "group_email", null) != null ||
        lookup(config, "group_key", null) != null
      )
      is_ou_policy = (
        lookup(config, "ou_path", null) != null ||
        lookup(config, "ou_key", null) != null
      )

      # Resolve group ID if this is a group policy
      resolved_group_id = (
        # If group_email is provided, look up the group ID from the data source
        lookup(config, "group_email", null) != null ?
        data.googleworkspace_group.group_lookup[config.group_email].id :
        # If group_key is provided, look it up from the groups_map variable
        lookup(config, "group_key", null) != null ?
        lookup(var.groups_map, config.group_key, null) != null ?
        var.groups_map[config.group_key].id :
        null :
        null
      )

      # Resolve OU ID if this is an OU policy
      resolved_ou_id = (
        # If ou_path is provided, look it up from the data source
        lookup(config, "ou_path", null) != null ?
        data.googleworkspace_org_unit.ou_lookup[config.ou_path].id :
        # If ou_key is provided, look it up from the ou_map variable
        lookup(config, "ou_key", null) != null ?
        lookup(var.ou_map, config.ou_key, null) != null ?
        var.ou_map[config.ou_key].id :
        null :
        null
      )
    })
  }

  # Separate policies by type
  group_policies = {
    for key, config in local.policies_with_resolved_targets :
    key => config
    if config.is_group_policy && !config.is_ou_policy
  }

  ou_policies = {
    for key, config in local.policies_with_resolved_targets :
    key => config
    if config.is_ou_policy && !config.is_group_policy
  }

  # Validate that all policies have a valid target reference
  invalid_group_policies = [
    for key, config in local.group_policies :
    key if config.resolved_group_id == null
  ]

  invalid_ou_policies = [
    for key, config in local.ou_policies :
    key if config.resolved_ou_id == null
  ]

  # Policies that have both group and OU specified (invalid)
  conflicting_policies = [
    for key, config in local.policies_with_resolved_targets :
    key if config.is_group_policy && config.is_ou_policy
  ]

  # Policies that have neither group nor OU specified (invalid)
  missing_target_policies = [
    for key, config in local.policies_with_resolved_targets :
    key if !config.is_group_policy && !config.is_ou_policy
  ]

  # ============================================================================
  # IMPORT SUPPORT
  # ============================================================================
  # Compute import IDs for policies with import: true in their YAML config.
  # These are used by the import {} blocks below to bring existing Google Admin
  # policies into Terraform state.

  ou_policies_to_import = {
    for key, config in local.ou_policies :
    key => {
      import_id = (
        length(lookup(config, "additional_target_keys", [])) > 0 ?
        join("/", [
          config.resolved_ou_id,
          join("&", [for k in lookup(config, "additional_target_keys", []) : "${k.target_key}=${k.target_value}"]),
          join(",", [for p in config.policies : p.schema_name])
        ]) :
        join("/", [
          config.resolved_ou_id,
          join(",", [for p in config.policies : p.schema_name])
        ])
      )
    }
    if lookup(config, "import", false) == true
  }

  group_policies_to_import = {
    for key, config in local.group_policies :
    key => {
      import_id = join("/", [
        config.resolved_group_id,
        join(",", [for p in config.policies : p.schema_name])
      ])
    }
    if lookup(config, "import", false) == true
  }
}

# Validation using terraform_data with preconditions - fails at PLAN time
resource "terraform_data" "validate_group_references" {
  lifecycle {
    precondition {
      condition     = length(local.invalid_group_policies) == 0
      error_message = <<-EOT
        INVALID GROUP REFERENCES:
        The following policies reference groups that don't exist in groups_map:
        ${join(", ", local.invalid_group_policies)}

        Resolution: Ensure the group_key exists in the groups module output or use group_email instead.
      EOT
    }
  }
}

resource "terraform_data" "validate_ou_references" {
  lifecycle {
    precondition {
      condition     = length(local.invalid_ou_policies) == 0
      error_message = <<-EOT
        INVALID OU REFERENCES:
        The following policies reference OUs that don't exist:
        ${join(", ", local.invalid_ou_policies)}

        Resolution: Ensure the ou_key exists in the ou_map or use a valid ou_path.
      EOT
    }
  }
}

resource "terraform_data" "validate_single_target" {
  lifecycle {
    precondition {
      condition     = length(local.conflicting_policies) == 0
      error_message = <<-EOT
        CONFLICTING TARGET SPECIFICATIONS:
        The following policies have both group AND OU targets specified (only one allowed):
        ${join(", ", local.conflicting_policies)}

        Resolution: Specify either a group target OR an OU target, not both.
      EOT
    }
  }
}

resource "terraform_data" "validate_has_target" {
  lifecycle {
    precondition {
      condition     = length(local.missing_target_policies) == 0
      error_message = <<-EOT
        MISSING TARGET SPECIFICATIONS:
        The following policies have no target (group or OU) specified:
        ${join(", ", local.missing_target_policies)}

        Resolution: Add group_key, group_email, ou_key, or ou_path to each policy.
      EOT
    }
  }
}

# Create Chrome group policies
resource "googleworkspace_chrome_group_policy" "group_policy" {
  for_each = local.group_policies

  group_id = each.value.resolved_group_id

  # Create policy blocks for each policy in the configuration
  # Uses resolved schema values that have asset references replaced with URLs
  dynamic "policies" {
    for_each = local.resolve_schema_values[each.key]
    content {
      schema_name = policies.value.schema_name
      schema_values = {
        for k, v in policies.value.schema_values :
        k => jsonencode(v)
      }
    }
  }

  # Add additional_target_keys if provided
  dynamic "additional_target_keys" {
    for_each = lookup(each.value, "additional_target_keys", [])
    content {
      target_key   = additional_target_keys.value.target_key
      target_value = additional_target_keys.value.target_value
    }
  }

  depends_on = [
    terraform_data.validate_group_references,
    terraform_data.validate_ou_references,
    terraform_data.validate_single_target,
    terraform_data.validate_has_target,
  ]
}

# Create Chrome OU policies
resource "googleworkspace_chrome_policy" "ou_policy" {
  for_each = local.ou_policies

  org_unit_id = each.value.resolved_ou_id

  # Create policy blocks for each policy in the configuration
  # Uses resolved schema values that have asset references replaced with URLs
  dynamic "policies" {
    for_each = local.resolve_schema_values[each.key]
    content {
      schema_name = policies.value.schema_name
      schema_values = {
        for k, v in policies.value.schema_values :
        k => jsonencode(v)
      }
    }
  }

  # Add additional_target_keys if provided
  dynamic "additional_target_keys" {
    for_each = lookup(each.value, "additional_target_keys", [])
    content {
      target_key   = additional_target_keys.value.target_key
      target_value = additional_target_keys.value.target_value
    }
  }

  depends_on = [
    terraform_data.validate_group_references,
    terraform_data.validate_ou_references,
    terraform_data.validate_single_target,
    terraform_data.validate_has_target,
  ]
}
