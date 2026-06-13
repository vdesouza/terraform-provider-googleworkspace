# Read all YAML files from the config directory
locals {
  # Get all YAML files in the directory
  yaml_files = fileset(var.yaml_config_path, "**/*.{yml,yaml}")

  # Parse all YAML files and merge them into a single map
  raw_groups = merge([
    for file in local.yaml_files : {
      for group_key, group_config in try(yamldecode(file("${var.yaml_config_path}/${file}")).groups, {}) :
      group_key => merge(group_config, {
        _source_file = file
      })
    }
  ]...)

  # ============================================================================
  # VARIABLE RESOLUTION
  # ============================================================================
  # Resolve {varname} references in group configurations.
  # Variables are defined in config/variables/ and passed via variables_map.

  # Build lookup map for variable references
  # Key: "{varname}", Value: variable value (string or list)
  variable_lookup = {
    for var_name, var_value in var.variables_map :
    "{${var_name}}" => var_value
  }

  # Helper to resolve a list with potential variable references
  # If an item is "{varname}" that resolves to a list, flatten it
  resolve_list = {
    for var_ref, var_value in local.variable_lookup :
    var_ref => var_value
  }

  # Resolve variables in group configurations
  # Handles email, user_members, user_managers, user_owners, group_members fields
  groups_with_resolved_variables = {
    for group_key, config in local.raw_groups :
    group_key => merge(config, {
      # Resolve email if it's a variable reference
      email = lookup(local.variable_lookup, config.email, config.email)

      # Resolve user_members list - flatten any variable references that expand to lists
      user_members = lookup(config, "user_members", null) != null ? flatten([
        for item in config.user_members :
        try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
      ]) : []

      # Resolve user_managers list
      user_managers = lookup(config, "user_managers", null) != null ? flatten([
        for item in config.user_managers :
        try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
      ]) : []

      # Resolve user_owners list
      user_owners = lookup(config, "user_owners", null) != null ? flatten([
        for item in config.user_owners :
        try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
      ]) : []

      # Resolve group_members list
      group_members = lookup(config, "group_members", null) != null ? flatten([
        for item in config.group_members :
        try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
      ]) : []
    })
  }

  # Separate groups into static and dynamic based on presence of query or org_units
  dynamic_groups = {
    for key, config in local.groups_with_resolved_variables :
    key => config
    if lookup(config, "query", null) != null || lookup(config, "org_units", null) != null
  }

  static_groups = {
    for key, config in local.groups_with_resolved_variables :
    key => config
    if lookup(config, "query", null) == null && lookup(config, "org_units", null) == null
  }

  # Extract all unique org unit paths from all dynamic groups
  all_org_unit_paths = distinct(flatten([
    for key, config in local.dynamic_groups :
    lookup(config, "org_units", [])
  ]))
}

# Fetch org unit data for all org units referenced in the configs
data "googleworkspace_org_unit" "ou_lookup" {
  for_each      = toset(local.all_org_unit_paths)
  org_unit_path = each.value
}

locals {
  # Build org unit queries for each dynamic group
  dynamic_group_queries = {
    for key, config in local.dynamic_groups :
    key => (
      # If both query and org_units are provided, AND them together
      lookup(config, "query", null) != null && lookup(config, "org_units", null) != null ?
      "(${config.query}) && (${join(" || ", [
        for ou_path in config.org_units :
        "(user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('${trimprefix(data.googleworkspace_org_unit.ou_lookup[ou_path].id, "id:")}')))"
      ])})" :
      # If only query is provided
      lookup(config, "query", null) != null ? config.query :
      # If only org_units are provided
      lookup(config, "org_units", null) != null ? join(" || ", [
        for ou_path in config.org_units :
        "(user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('${trimprefix(data.googleworkspace_org_unit.ou_lookup[ou_path].id, "id:")}')))"
      ]) : ""
    )
  }
}

# Create dynamic groups (with queries)
resource "googleworkspace_group_dynamic" "dynamic" {
  for_each = local.dynamic_groups

  email        = each.value.email
  display_name = lookup(each.value, "display_name", each.value.email)
  description  = lookup(each.value, "description", "")

  # Build the query - if custom query is provided, use it; otherwise build from org_units
  query = local.dynamic_group_queries[each.key] != "" ? (
    lookup(each.value, "suspended_filter", true) ?
    "(${local.dynamic_group_queries[each.key]}) && user.suspended == false" :
    local.dynamic_group_queries[each.key]
  ) : ""

  security_group = lookup(each.value, "security_group", false)
  locked         = lookup(each.value, "locked", false)

  # Add custom labels if provided
  labels = lookup(each.value, "labels", {})
}

# Create static groups (with members)
resource "googleworkspace_group" "static" {
  for_each = local.static_groups

  email          = each.value.email
  name           = lookup(each.value, "display_name", each.value.email)
  description    = lookup(each.value, "description", "")
  security_group = lookup(each.value, "security_group", false)

  # Add aliases if provided
  aliases = lookup(each.value, "aliases", [])
}

# Add members (users and groups) to static groups
resource "googleworkspace_group_member" "static_members" {
  # Ensure all groups are created before adding any members
  depends_on = [
    googleworkspace_group.static,
    googleworkspace_group_dynamic.dynamic
  ]

  for_each = merge([
    for group_key, group_config in local.static_groups : merge(
      # User members with MEMBER role
      {
        for idx, user_email in lookup(group_config, "user_members", []) :
        "${group_key}_user_member_${idx}" => {
          group_id = googleworkspace_group.static[group_key].id
          email    = user_email
          role     = "MEMBER"
          type     = "USER"
        }
      },
      # User managers with MANAGER role
      {
        for idx, user_email in lookup(group_config, "user_managers", []) :
        "${group_key}_user_manager_${idx}" => {
          group_id = googleworkspace_group.static[group_key].id
          email    = user_email
          role     = "MANAGER"
          type     = "USER"
        }
      },
      # User owners with OWNER role
      {
        for idx, user_email in lookup(group_config, "user_owners", []) :
        "${group_key}_user_owner_${idx}" => {
          group_id = googleworkspace_group.static[group_key].id
          email    = user_email
          role     = "OWNER"
          type     = "USER"
        }
      },
      # Group members with MEMBER role
      {
        for idx, group_email in lookup(group_config, "group_members", []) :
        "${group_key}_group_member_${idx}" => {
          group_id = googleworkspace_group.static[group_key].id
          email    = group_email
          role     = "MEMBER"
          type     = "GROUP"
        }
      }
    )
  ]...)

  group_id = each.value.group_id
  email    = each.value.email
  role     = each.value.role
  type     = each.value.type
}
