# Read all YAML files from the config directory
locals {
  # Get all YAML files in the directory
  yaml_files = fileset(var.yaml_config_path, "**/*.{yml,yaml}")

  # Parse each YAML file individually
  parsed_yaml_files = {
    for file in local.yaml_files :
    file => try(yamldecode(file("${var.yaml_config_path}/${file}")), {})
  }

  # ============================================================================
  # POLICY ORDERING PARSING
  # ============================================================================

  # Find all files that define default_policy_ordering
  files_with_default_policy_ordering = [
    for file, content in local.parsed_yaml_files :
    file
    if lookup(content, "default_policy_ordering", null) != null
  ]

  # Validation: Ensure only one file defines default_policy_ordering
  duplicate_default_policy_ordering_error = length(local.files_with_default_policy_ordering) > 1 ? [
    "Multiple YAML files define 'default_policy_ordering'. Only one file should define the default ordering. Found in: ${join(", ", local.files_with_default_policy_ordering)}"
  ] : []

  # Parse all YAML files and merge the ordering configuration
  raw_config = merge([
    for file, content in local.parsed_yaml_files :
    content
  ]...)

  # ============================================================================
  # VARIABLE RESOLUTION
  # ============================================================================
  # Resolve {varname} references in ordering configurations.
  # Variables are defined in config/variables/ and passed via variables_map.

  # Build lookup map for variable references
  # Key: "{varname}", Value: variable value (string or list)
  variable_lookup = {
    for var_name, var_value in var.variables_map :
    "{${var_name}}" => var_value
  }

  # Helper to resolve ordering list - handles both:
  # 1. Entire list is a variable reference: "{admin_emails}" -> list
  # 2. Individual items are variable references that expand to lists
  resolve_ordering_list = {
    for ordering_key, ordering_value in {
      "default_policy"    = lookup(local.raw_config, "default_policy_ordering", [])
      "default_extension" = lookup(local.raw_config, "default_extension_ordering", [])
    } :
    ordering_key => (
      # Check if the ordering is a single variable reference (string that resolves to list)
      try(tostring(ordering_value), null) != null ? (
        lookup(local.variable_lookup, ordering_value, [ordering_value])
        ) : (
        # Otherwise it's already a list, resolve any variable references within
        flatten([
          for item in ordering_value :
          try(tostring(item), null) != null ? (
            # If item is a variable reference that resolves to a list, flatten it
            try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
          ) : [item]
        ])
      )
    )
  }

  # Extract the default policy ordering (list of group emails in priority order)
  # With variable resolution applied
  default_policy_ordering_emails = local.resolve_ordering_list["default_policy"]

  # ============================================================================
  # EXTENSION ORDERING PARSING
  # ============================================================================

  # Find all files that define default_extension_ordering
  # Note: default_extension_ordering can ONLY be defined in group_priority YAML files
  # (similar to default_policy_ordering). Extension YAML files can only define extension_orderings.
  files_with_default_extension_ordering = [
    for file, content in local.parsed_yaml_files :
    file
    if lookup(content, "default_extension_ordering", null) != null
  ]

  # Validation: Ensure only one file defines default_extension_ordering
  duplicate_default_extension_ordering_error = length(local.files_with_default_extension_ordering) > 1 ? [
    "Multiple YAML files define 'default_extension_ordering'. Only one file should define the default ordering. Found in: ${join(", ", local.files_with_default_extension_ordering)}"
  ] : []

  # Extract the default extension ordering (list of group emails in priority order)
  # With variable resolution applied
  default_extension_ordering_emails = local.resolve_ordering_list["default_extension"]

  # Extract extension-specific orderings from local YAML files with source info
  local_extension_orderings_by_file = {
    for file, content in local.parsed_yaml_files :
    file => lookup(content, "extension_orderings", {})
    if lookup(content, "extension_orderings", null) != null
  }

  # Build local extension_orderings with source for duplicate detection
  local_extension_orderings_with_source = merge([
    for file, orderings in local.local_extension_orderings_by_file : {
      for ext_id, ordering in orderings :
      ext_id => {
        source_file = "group_priority/${file}"
        ordering    = ordering
      }
    }
  ]...)

  # Extract extension-specific orderings from local YAML (flat map)
  # With variable resolution applied
  local_extension_orderings_raw = lookup(local.raw_config, "extension_orderings", {})
  local_extension_orderings = {
    for ext_id, ordering in local.local_extension_orderings_raw :
    ext_id => (
      # Check if the ordering is a single variable reference (string that resolves to list)
      try(tostring(ordering), null) != null ? (
        lookup(local.variable_lookup, ordering, [ordering])
        ) : (
        # Otherwise it's already a list, resolve any variable references within
        flatten([
          for item in ordering :
          try(tostring(item), null) != null ? (
            try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
          ) : [item]
        ])
      )
    )
  }

  # Combine all extension_orderings with source info for duplicate detection
  all_extension_orderings_with_source = merge(
    local.local_extension_orderings_with_source,
    {
      for ext_id, config in var.external_extension_orderings_with_source :
      ext_id => {
        source_file = "extensions/${config.source_file}"
        ordering    = config.ordering
      }
    }
  )

  # Detect duplicate extension_orderings (same extension defined in multiple files)
  extension_ordering_sources = {
    for ext_id in distinct(concat(
      keys(local.local_extension_orderings_with_source),
      keys(var.external_extension_orderings_with_source)
    )) :
    ext_id => compact([
      contains(keys(local.local_extension_orderings_with_source), ext_id) ?
      local.local_extension_orderings_with_source[ext_id].source_file : null,
      contains(keys(var.external_extension_orderings_with_source), ext_id) ?
      "extensions/${var.external_extension_orderings_with_source[ext_id].source_file}" : null
    ])
  }

  # Find extensions with orderings defined in multiple places
  duplicate_extension_orderings = {
    for ext_id, sources in local.extension_ordering_sources :
    ext_id => sources
    if length(sources) > 1
  }

  # Validation error for duplicate extension_orderings
  duplicate_extension_orderings_error = length(local.duplicate_extension_orderings) > 0 ? [
    for ext_id, sources in local.duplicate_extension_orderings :
    "Extension '${ext_id}' has ordering defined in multiple files: ${join(", ", sources)}. Each extension can only have one ordering definition."
  ] : []

  # Merge local and external extension_orderings (local takes precedence if no duplicates)
  extension_orderings = merge(
    var.external_extension_orderings,
    local.local_extension_orderings
  )

  # Extract policy-specific orderings from local YAML files with source info
  local_policy_orderings_by_file = {
    for file, content in local.parsed_yaml_files :
    file => lookup(content, "policy_orderings", {})
    if lookup(content, "policy_orderings", null) != null
  }

  # Build local policy_orderings with source for duplicate detection
  local_policy_orderings_with_source = merge([
    for file, orderings in local.local_policy_orderings_by_file : {
      for schema, ordering in orderings :
      schema => {
        source_file = "group_priority/${file}"
        ordering    = ordering
      }
    }
  ]...)

  # Extract policy-specific orderings from local YAML (flat map)
  # With variable resolution applied
  local_policy_orderings_raw = lookup(local.raw_config, "policy_orderings", {})
  local_policy_orderings = {
    for schema, ordering in local.local_policy_orderings_raw :
    schema => (
      # Check if the ordering is a single variable reference (string that resolves to list)
      try(tostring(ordering), null) != null ? (
        lookup(local.variable_lookup, ordering, [ordering])
        ) : (
        # Otherwise it's already a list, resolve any variable references within
        flatten([
          for item in ordering :
          try(tostring(item), null) != null ? (
            try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
          ) : [item]
        ])
      )
    )
  }

  # Combine all policy_orderings with source info for duplicate detection
  # Prefix local sources with "group_priority/" to distinguish from external sources
  all_policy_orderings_with_source = merge(
    local.local_policy_orderings_with_source,
    {
      for schema, config in var.external_policy_orderings_with_source :
      schema => {
        source_file = "policies/${config.source_file}"
        ordering    = config.ordering
      }
    }
  )

  # Detect duplicate policy_orderings (same policy defined in multiple files)
  # Group policy schemas by their source files
  policy_ordering_sources = {
    for schema in distinct(concat(
      keys(local.local_policy_orderings_with_source),
      keys(var.external_policy_orderings_with_source)
    )) :
    schema => compact([
      contains(keys(local.local_policy_orderings_with_source), schema) ?
      local.local_policy_orderings_with_source[schema].source_file : null,
      contains(keys(var.external_policy_orderings_with_source), schema) ?
      "policies/${var.external_policy_orderings_with_source[schema].source_file}" : null
    ])
  }

  # Find policies with orderings defined in multiple places
  duplicate_policy_orderings = {
    for schema, sources in local.policy_ordering_sources :
    schema => sources
    if length(sources) > 1
  }

  # Validation error for duplicate policy_orderings
  duplicate_policy_orderings_error = length(local.duplicate_policy_orderings) > 0 ? [
    for schema, sources in local.duplicate_policy_orderings :
    "Policy '${schema}' has ordering defined in multiple files: ${join(", ", sources)}. Each policy can only have one ordering definition."
  ] : []

  # Merge local and external policy_orderings (local takes precedence if no duplicates)
  # This uses the external (policies YAML) orderings with local (group_priority) as override
  policy_orderings = merge(
    var.external_policy_orderings,
    local.local_policy_orderings
  )

  # Convert default policy ordering emails to group IDs using groups_map
  default_policy_ordering_ids = [
    for email in local.default_policy_ordering_emails :
    var.groups_map[email].id
    if contains(keys(var.groups_map), email)
  ]

  # Build a map of group email -> group ID for quick lookup
  # Merge from two sources:
  # 1. groups_map: groups managed by the groups module
  # 2. policies_map: groups referenced in policies (includes external groups by email)

  # First, get unique group email/id pairs from policies_map
  policy_groups = {
    for policy_key, policy_config in var.policies_map :
    policy_config.group_email => policy_config.group_id
    if policy_config.group_email != null
  }

  group_email_to_id = merge(
    {
      for _key, group in var.groups_map :
      group.email => group.id
    },
    local.policy_groups
  )

  # Get all unique policy schemas from the policies being managed
  all_policy_schemas = distinct(flatten([
    for policy_key, policy_config in var.policies_map :
    [for p in policy_config.policies : p.schema_name]
  ]))

  # Build a map of policy_schema -> list of group IDs that have this policy applied
  # This comes from the policies module output
  groups_per_policy = {
    for schema in local.all_policy_schemas :
    schema => distinct([
      for policy_key, policy_config in var.policies_map :
      policy_config.group_id
      if policy_config.group_id != null && contains([for p in policy_config.policies : p.schema_name], schema)
    ])
  }

  # Build a map of policy_schema -> list of group emails that have this policy applied
  groups_emails_per_policy = {
    for schema in local.all_policy_schemas :
    schema => distinct([
      for policy_key, policy_config in var.policies_map :
      policy_config.group_email
      if policy_config.group_email != null && contains([for p in policy_config.policies : p.schema_name], schema)
    ])
  }

  # ============================================================================
  # UNMANAGED GROUP RESOLUTION
  # ============================================================================
  # Groups in ordering configs that aren't managed by Terraform (e.g., assigned
  # in Google Admin console directly) need to be looked up via data source.

  # All unique emails referenced in any ordering
  all_ordering_emails = distinct(concat(
    local.default_policy_ordering_emails,
    flatten(values(local.policy_orderings)),
    local.default_extension_ordering_emails,
    flatten(values(local.extension_orderings))
  ))

  # Get unique group email/id pairs from extensions_map (needed before group_email_to_id_with_extensions)
  extension_groups = {
    for email, ids in {
      for ext_key, ext_config in var.extensions_map :
      ext_config.group_email => ext_config.group_id...
      if ext_config.group_email != null
    } :
    email => ids[0]
  }

  # Combined map (policies + extensions) — used to detect which ordering emails are already known
  group_email_to_id_with_extensions = merge(
    local.group_email_to_id,
    local.extension_groups
  )

  # Emails in orderings that cannot be resolved from managed groups
  unresolved_ordering_emails = [
    for email in local.all_ordering_emails :
    email
    if !contains(keys(local.group_email_to_id_with_extensions), email)
  ]

  # ============================================================================
  # EXTENSION GROUP ORDERING COMPUTATION
  # ============================================================================

  # Get all unique extension app_ids from the extensions being managed
  all_extension_app_ids = distinct([
    for ext_key, ext_config in var.extensions_map :
    ext_config.app_id
    if ext_config.app_id != null
  ])

  # Build a map of app_id -> list of group IDs that have this extension applied
  groups_per_extension = {
    for app_id in local.all_extension_app_ids :
    app_id => distinct([
      for ext_key, ext_config in var.extensions_map :
      ext_config.group_id
      if ext_config.group_id != null && ext_config.app_id == app_id
    ])
  }

  # Build a map of app_id -> list of group emails that have this extension applied
  groups_emails_per_extension = {
    for app_id in local.all_extension_app_ids :
    app_id => distinct([
      for ext_key, ext_config in var.extensions_map :
      ext_config.group_email
      if ext_config.group_email != null && ext_config.app_id == app_id
    ])
  }

  # Strip prefix from app_id for ordering lookup (orderings use extension ID without prefix)
  strip_app_id_prefix = {
    for app_id in local.all_extension_app_ids :
    app_id => replace(replace(replace(app_id, "chrome:", ""), "android:", ""), "web:", "")
  }

  # Extended email-to-id maps that include unmanaged groups resolved via data source
  ordering_group_email_to_id = {
    for email, group in data.googleworkspace_group.ordering_groups :
    email => group.id
  }

  group_email_to_id_complete = merge(
    local.group_email_to_id,
    local.ordering_group_email_to_id
  )

  group_email_to_id_with_extensions_complete = merge(
    local.group_email_to_id_with_extensions,
    local.ordering_group_email_to_id
  )

  # Loose (first-pass) ordering computation — includes all resolvable unmanaged groups.
  # Used only to determine which schemas need an API query; NOT used for the actual resource.
  policy_group_orderings_loose = {
    for schema in local.all_policy_schemas :
    schema => {
      ordering_emails = lookup(local.policy_orderings, schema, local.default_policy_ordering_emails)

      filtered_group_ids = [
        for email in lookup(local.policy_orderings, schema, local.default_policy_ordering_emails) :
        local.group_email_to_id_complete[email]
        if contains(keys(local.group_email_to_id_complete), email) && (
          contains(local.groups_emails_per_policy[schema], email) ||
          contains(keys(local.ordering_group_email_to_id), email)
        )
      ]

      # Groups that have the policy but are NOT in any ordering (validation will catch this)
      groups_without_ordering = [
        for email in local.groups_emails_per_policy[schema] :
        email
        if !contains(lookup(local.policy_orderings, schema, local.default_policy_ordering_emails), email)
      ]
    }
  }

  # Validation: Collect all groups that have policies but are not in any ordering
  groups_missing_from_ordering_errors = flatten([
    for schema, config in local.policy_group_orderings_loose :
    [
      for email in config.groups_without_ordering :
      "Group '${email}' has policy '${schema}' applied but is not present in the ordering definition"
    ]
  ])

  # Schemas that need an API query (loose filter — superset of final policies_needing_ordering).
  # Drives the api_policy_ordering data source; must be computed before the strict pass.
  policies_needing_api_query = {
    for schema, config in local.policy_group_orderings_loose :
    schema => config
    if length(config.filtered_group_ids) >= 2
  }

  # Loose (first-pass) extension ordering — same pattern as policy_group_orderings_loose.
  extension_group_orderings_loose = {
    for app_id in local.all_extension_app_ids :
    app_id => {
      extension_id_no_prefix = local.strip_app_id_prefix[app_id]
      ordering_emails        = lookup(local.extension_orderings, local.strip_app_id_prefix[app_id], local.default_extension_ordering_emails)

      filtered_group_ids = [
        for email in lookup(local.extension_orderings, local.strip_app_id_prefix[app_id], local.default_extension_ordering_emails) :
        local.group_email_to_id_with_extensions_complete[email]
        if contains(keys(local.group_email_to_id_with_extensions_complete), email) && (
          contains(local.groups_emails_per_extension[app_id], email) ||
          contains(keys(local.ordering_group_email_to_id), email)
        )
      ]

      # Groups that have the extension but are NOT in any ordering (validation will catch this)
      groups_without_ordering = [
        for email in local.groups_emails_per_extension[app_id] :
        email
        if email != null && !contains(lookup(local.extension_orderings, local.strip_app_id_prefix[app_id], local.default_extension_ordering_emails), email)
      ]

      all_group_ids = local.groups_per_extension[app_id]
    }
  }

  # Validation: Collect all groups that have extensions but are not in any ordering
  groups_missing_from_extension_ordering_errors = flatten([
    for app_id, config in local.extension_group_orderings_loose :
    [
      for email in config.groups_without_ordering :
      "Group '${email}' has extension '${app_id}' applied but is not present in the ordering (default_extension_ordering or extension_orderings['${local.strip_app_id_prefix[app_id]}'])"
    ]
    if length(config.groups_without_ordering) > 0
  ])

  # Extensions that need an API query (loose filter — superset of final extensions_needing_ordering).
  extensions_needing_api_query = {
    for app_id, config in local.extension_group_orderings_loose :
    app_id => config
    if length(config.filtered_group_ids) >= 2
  }

  # Combine all validation errors (static checks only — values must be known at plan time)
  # API mismatch validation is handled via lifecycle.precondition on the ordering resources,
  # which can defer evaluation to apply time if API values are not yet known during plan.
  validation_errors = concat(
    local.duplicate_default_policy_ordering_error,
    local.duplicate_default_extension_ordering_error,
    local.duplicate_policy_orderings_error,
    local.duplicate_extension_orderings_error,
    local.groups_missing_from_ordering_errors,
    local.groups_missing_from_extension_ordering_errors
  )
}

# ============================================================================
# DATA SOURCES
# ============================================================================

# Look up unmanaged groups referenced in ordering configs by email.
# These are groups assigned to policies/extensions in Google Admin but not
# managed by Terraform. If the email doesn't exist, this data source fails
# the plan with a clear error (catches typos).
data "googleworkspace_group" "ordering_groups" {
  for_each = toset(local.unresolved_ordering_emails)
  email    = each.value
}

# Look up the root OU for the current Google Workspace account.
# Extension group priority ordering requires a target_resource (OU) even for
# group-based policies, because the Chrome Policy API scopes app-specific
# orderings to an OU + app_id combination.
data "googleworkspace_org_unit" "root" {
  org_unit_path = "/"
}

# Query the API for the current group priority ordering for each policy schema.
# Used to detect groups that have the policy applied in Google Admin but are
# not accounted for in our Terraform ordering config.
data "googleworkspace_chrome_policy_group_priority_ordering" "api_policy_ordering" {
  for_each      = local.policies_needing_api_query
  policy_schema = each.key
}

# Query the API for the current group priority ordering for each extension.
data "googleworkspace_chrome_policy_group_priority_ordering" "api_extension_ordering" {
  for_each = local.extensions_needing_api_query

  policy_schema = "chrome.users.apps.InstallType"

  policy_target_key {
    target_resource = "orgunits/${data.googleworkspace_org_unit.root.id}"
    additional_target_key_names = {
      "app_id" = each.key
    }
  }
}

# ============================================================================
# STRICT ORDERING COMPUTATION (second pass, uses API data)
# ============================================================================
# Some groups in the ordering config are not created by this repo (e.g.,
# active-employees, active-cs-partners).  They may have some Chrome policies
# applied via this repo but not others.  Condition (a) in the loose filter
# already handles the schemas where they DO have the policy (group_email is in
# groups_emails_per_policy).  Condition (b) — "is an unmanaged group" — is too
# broad: it includes them for every schema, even ones they don't have applied,
# which causes the Chrome Policy API to return 400 "Request contains an invalid
# argument".
#
# The strict pass replaces condition (b) with an API-data check: an unmanaged
# group is only included in a schema's ordering if the Chrome Policy API already
# lists them in that ordering, which proves they have the policy applied for
# that specific schema.
locals {
  # Map schema → list of group IDs currently in the API ordering (empty if none exists).
  api_policy_ordering_group_ids = {
    for schema, ordering in data.googleworkspace_chrome_policy_group_priority_ordering.api_policy_ordering :
    schema => ordering.exists ? ordering.group_ids : []
  }

  # Map app_id → list of group IDs currently in the API extension ordering.
  api_extension_ordering_group_ids = {
    for app_id, ordering in data.googleworkspace_chrome_policy_group_priority_ordering.api_extension_ordering :
    app_id => ordering.exists ? ordering.group_ids : []
  }

  # Strict policy orderings: unmanaged groups only included if they appear in
  # the API's existing ordering for that schema.
  policy_group_orderings = {
    for schema, config in local.policy_group_orderings_loose :
    schema => {
      ordering_emails         = config.ordering_emails
      groups_without_ordering = config.groups_without_ordering
      filtered_group_ids = [
        for email in config.ordering_emails :
        local.group_email_to_id_complete[email]
        if contains(keys(local.group_email_to_id_complete), email) && (
          contains(local.groups_emails_per_policy[schema], email) ||
          (
            contains(keys(local.ordering_group_email_to_id), email) &&
            contains(lookup(local.api_policy_ordering_group_ids, schema, []), local.group_email_to_id_complete[email])
          )
        )
      ]
    }
  }

  policies_needing_ordering = {
    for schema, config in local.policy_group_orderings :
    schema => config
    if length(config.filtered_group_ids) >= 2
  }

  # Strict extension orderings: same pattern.
  extension_group_orderings = {
    for app_id, config in local.extension_group_orderings_loose :
    app_id => {
      extension_id_no_prefix  = config.extension_id_no_prefix
      ordering_emails         = config.ordering_emails
      groups_without_ordering = config.groups_without_ordering
      all_group_ids           = config.all_group_ids
      filtered_group_ids = [
        for email in config.ordering_emails :
        local.group_email_to_id_with_extensions_complete[email]
        if contains(keys(local.group_email_to_id_with_extensions_complete), email) && (
          contains(local.groups_emails_per_extension[app_id], email) ||
          (
            contains(keys(local.ordering_group_email_to_id), email) &&
            contains(lookup(local.api_extension_ordering_group_ids, app_id, []), local.group_email_to_id_with_extensions_complete[email])
          )
        )
      ]
    }
  }

  extensions_needing_ordering = {
    for app_id, config in local.extension_group_orderings :
    app_id => config
    if length(config.filtered_group_ids) >= 2
  }
}

# Validation resource - fails during plan if any groups are missing from ordering
resource "null_resource" "validate_orderings" {
  count = length(local.validation_errors) > 0 ? 1 : 0

  lifecycle {
    precondition {
      condition     = length(local.validation_errors) == 0
      error_message = "Group priority ordering validation failed:\n${join("\n", local.validation_errors)}"
    }
  }
}

# Create group priority ordering for each policy schema that has multiple groups
resource "googleworkspace_chrome_policy_group_priority_ordering" "ordering" {
  for_each = local.policies_needing_ordering

  policy_schema = each.key

  # policy_target_key is only required for app-specific policies (chrome.users.apps.*)
  # For regular user policies (chrome.users.*), omit this block entirely
  # dynamic "policy_target_key" {
  #   for_each = var.app_id != null ? [1] : []
  #   content {
  #     target_resource = var.org_unit_id
  #     additional_target_key_names = {
  #       "app_id" = var.app_id
  #     }
  #   }
  # }

  group_ids = each.value.filtered_group_ids

  lifecycle {
    # Validate that no groups are assigned this policy in Google Admin outside of Terraform.
    # If exists=false the API returned 400 (no ordering yet) so there's nothing to check.
    # If values are unknown at plan time this precondition is deferred to apply time,
    # where it runs before the API call — preventing a confusing 400 error.
    precondition {
      condition = !data.googleworkspace_chrome_policy_group_priority_ordering.api_policy_ordering[each.key].exists || length([
        for id in data.googleworkspace_chrome_policy_group_priority_ordering.api_policy_ordering[each.key].group_ids :
        id if !contains(each.value.filtered_group_ids, id)
      ]) == 0
      error_message = "Policy '${each.key}': The following group IDs are assigned this policy in Google Admin but are not in the Terraform ordering: [${join(", ", [for id in data.googleworkspace_chrome_policy_group_priority_ordering.api_policy_ordering[each.key].group_ids : id if !contains(each.value.filtered_group_ids, id)])}]. Look up these IDs in Google Admin (Directory > Groups), then add their emails to the ordering in your YAML config (default_policy_ordering or policy_orderings) or remove their policy assignments."
    }
  }

  depends_on = [null_resource.validate_orderings]
}

# Create group priority ordering for each extension that has multiple groups
resource "googleworkspace_chrome_policy_group_priority_ordering" "extension_ordering" {
  for_each = local.extensions_needing_ordering

  # Use InstallType as the policy schema for extension ordering
  policy_schema = "chrome.users.apps.InstallType"

  # Add the app_id as a target key to scope this ordering to the specific extension
  policy_target_key {
    target_resource = "orgunits/${data.googleworkspace_org_unit.root.id}"
    additional_target_key_names = {
      "app_id" = each.key
    }
  }

  group_ids = each.value.filtered_group_ids

  lifecycle {
    # Validate that no groups are assigned this extension in Google Admin outside of Terraform.
    # If exists=false the API returned 400 (no ordering yet) so there's nothing to check.
    # If values are unknown at plan time this precondition is deferred to apply time,
    # where it runs before the API call — preventing a confusing 400 error.
    precondition {
      condition = !data.googleworkspace_chrome_policy_group_priority_ordering.api_extension_ordering[each.key].exists || length([
        for id in data.googleworkspace_chrome_policy_group_priority_ordering.api_extension_ordering[each.key].group_ids :
        id if !contains(each.value.filtered_group_ids, id)
      ]) == 0
      error_message = "Extension '${each.key}': The following group IDs are assigned this extension in Google Admin but are not in the Terraform ordering: [${join(", ", [for id in data.googleworkspace_chrome_policy_group_priority_ordering.api_extension_ordering[each.key].group_ids : id if !contains(each.value.filtered_group_ids, id)])}]. Look up these IDs in Google Admin (Directory > Groups), then add their emails to the ordering in your YAML config (default_extension_ordering or extension_orderings) or remove their extension assignments."
    }
  }

  depends_on = [null_resource.validate_orderings]
}
