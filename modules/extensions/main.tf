# Read all YAML files from the config directory
locals {
  # Mapping from simplified field names to their schema names
  # This allows users to write simplified YAML without knowing schema names
  field_to_schema_map = {
    # PermissionsAndUrlAccess fields
    "blockedPermissions" = "chrome.users.apps.PermissionsAndUrlAccess"
    "allowedPermissions" = "chrome.users.apps.PermissionsAndUrlAccess"
    "blockedHosts"       = "chrome.users.apps.PermissionsAndUrlAccess"
    "allowedHosts"       = "chrome.users.apps.PermissionsAndUrlAccess"

    # ManagedConfiguration field
    "managedConfiguration" = "chrome.users.apps.ManagedConfiguration"

    # AccessToKeys field (was part of deprecated CertificateManagement)
    "allowAccessToKeys" = "chrome.users.apps.AccessToKeys"

    # EnterpriseChallenge field (split from deprecated CertificateManagement)
    "allowEnterpriseChallenge" = "chrome.users.apps.EnterpriseChallenge"

    # IncludeInChromeWebStoreCollection fields
    "includeInChromeWebStoreCollection" = "chrome.users.apps.IncludeInChromeWebStoreCollection"
    "includeInCollection"               = "chrome.users.apps.IncludeInChromeWebStoreCollection"
    "spotlightRecommended"              = "chrome.users.apps.IncludeInChromeWebStoreCollection"

    # DefaultLaunchContainer field
    "defaultLaunchContainer" = "chrome.users.apps.DefaultLaunchContainer"

    # SkipPrintConfirmation field
    "skipPrintConfirmation" = "chrome.users.apps.SkipPrintConfirmation"

  }

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

  # Parse each YAML file individually
  parsed_yaml_files = {
    for item in local.all_yaml_files :
    item.key => try(yamldecode(file("${item.path}/${item.file}")), {})
  }

  # ============================================================================
  # EXTENSION ORDERING PARSING
  # ============================================================================
  # Note: Extension-specific orderings are parsed here and passed to the group_priority module
  # which handles merging with group_priority YAML orderings and creates the ordering resources.
  #
  # IMPORTANT: default_extension_ordering must be defined in group_priority YAML files only
  # (similar to default_policy_ordering). Only extension_orderings can be defined here.

  # Extract extension-specific orderings from YAML files with source info
  local_extension_orderings_by_file = {
    for file, content in local.parsed_yaml_files :
    file => lookup(content, "extension_orderings", {})
    if lookup(content, "extension_orderings", null) != null
  }

  # Build extension_orderings with source for duplicate detection (to be passed to group_priority)
  local_extension_orderings_with_source = merge([
    for file, orderings in local.local_extension_orderings_by_file : {
      for ext_id, ordering in orderings :
      ext_id => {
        source_file = file
        ordering    = ordering
      }
    }
  ]...)

  # ============================================================================
  # VARIABLE RESOLUTION (must happen before extension_groups expansion)
  # ============================================================================
  # Resolve {varname} references in extension configurations.
  # Variables are defined in config/variables/ and passed via variables_map.

  # Build lookup map for variable references
  # Key: "{varname}", Value: variable value (string or list)
  variable_lookup = {
    for var_name, var_value in var.variables_map :
    "{${var_name}}" => var_value
  }

  # Parse individual extensions from all YAML files
  individual_extensions = merge([
    for file, content in local.parsed_yaml_files : {
      for extension_key, extension_config in lookup(content, "extensions", {}) :
      extension_key => merge(extension_config, {
        _source_file = file
      })
    }
  ]...)

  # Parse extension_groups and expand them into individual extension entries
  # extension_groups allows configuring multiple extensions with shared settings
  # The "extensions" field can be a variable reference like "{developer_tools_list}"
  expanded_extension_groups = merge([
    for file, content in local.parsed_yaml_files : merge([
      for group_key, group_config in lookup(content, "extension_groups", {}) : {
        # Resolve the extensions list - it could be a variable reference
        for idx, ext_id in(
          # Check if extensions is a string (variable reference) or already a list
          try(tostring(lookup(group_config, "extensions", [])), null) != null ? (
            # It's a string - look up as variable, default to empty list if not found
            try(tolist(lookup(local.variable_lookup, lookup(group_config, "extensions", ""), [])), [])
            ) : (
            # It's already a list - resolve any variable references within the list
            flatten([
              for item in lookup(group_config, "extensions", []) :
              try(tostring(item), null) != null ? (
                # If item is a variable reference that resolves to a list, flatten it
                try(tolist(lookup(local.variable_lookup, item, [item])), [lookup(local.variable_lookup, item, item)])
              ) : [item]
            ])
          )
        ) :
        # Generate unique key: group_key_extensionid
        "${group_key}_${ext_id}" => merge(
          # Remove the extensions list from group config and add the extension_id
          { for k, v in group_config : k => v if k != "extensions" },
          {
            # If the ID has a prefix (chrome:, android:, web:), use app_id field
            # so the module preserves the correct prefix. Otherwise treat as raw
            # extension_id (chrome extension).
            app_id       = can(regex("^(chrome|android|web):", ext_id)) ? ext_id : null
            extension_id = can(regex("^(chrome|android|web):", ext_id)) ? null : ext_id
            _source_file = file
            _from_group  = group_key
          }
        )
      }
    ]...)
  ]...)

  # Merge individual extensions with expanded extension groups
  # Individual extensions take precedence over group-expanded ones
  raw_extensions = merge(local.expanded_extension_groups, local.individual_extensions)

  # Resolve variables in extension configurations
  # Handles group_email, group_key, ou_path, ou_key fields
  extensions_with_resolved_variables = {
    for ext_key, config in local.raw_extensions :
    ext_key => merge(config, {
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

  # Helper function to determine the full app_id with prefix
  # Supports: extension_id, chrome_extension_id, android_app_id, web_app_url
  # Uses extensions_with_resolved_variables which has {varname} references resolved
  extensions_with_app_id = {
    for key, config in local.extensions_with_resolved_variables :
    key => merge(config, {
      # Determine the app_type and construct the full app_id
      app_type = (
        lookup(config, "extension_id", null) != null || lookup(config, "chrome_extension_id", null) != null ? "chrome" :
        lookup(config, "android_app_id", null) != null ? "android" :
        lookup(config, "web_app_url", null) != null ? "web" :
        lookup(config, "app_type", null) != null ? config.app_type :
        "chrome" # default to chrome
      )

      # Construct the full app_id with prefix
      full_app_id = (
        # Chrome extension
        lookup(config, "chrome_extension_id", null) != null ?
        "chrome:${config.chrome_extension_id}" :
        lookup(config, "extension_id", null) != null ?
        "chrome:${config.extension_id}" :
        # Android app
        lookup(config, "android_app_id", null) != null ?
        "android:${config.android_app_id}" :
        # Web app
        lookup(config, "web_app_url", null) != null ?
        "web:${config.web_app_url}" :
        # Explicit app_id (already has prefix)
        lookup(config, "app_id", null) != null ?
        config.app_id :
        null
      )

      # Determine if this is a group or OU extension
      is_group_extension = (
        lookup(config, "group_email", null) != null ||
        lookup(config, "group_key", null) != null
      )
      is_ou_extension = (
        lookup(config, "ou_path", null) != null ||
        lookup(config, "ou_key", null) != null
      )
    })
  }

  # Extract all unique OU paths referenced in the configs
  all_ou_paths = distinct(compact([
    for key, config in local.extensions_with_app_id :
    lookup(config, "ou_path", null)
  ]))

  # Extract all unique group emails that are directly referenced (not via group_key)
  all_group_emails = distinct(compact([
    for key, config in local.extensions_with_app_id :
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
  # Step 0: Resolve variables in configurations field
  # The configurations field can be:
  # 1. A string variable reference: "{config_var}" -> resolves to a list of configs
  # 2. A list (already resolved or literal)
  extensions_with_resolved_configs = {
    for key, config in local.extensions_with_app_id :
    key => merge(config, {
      configurations = (
        # Check if configurations is a string (variable reference)
        try(tostring(lookup(config, "configurations", [])), null) != null ? (
          # It's a string - look up as variable, default to empty list if not found
          try(tolist(lookup(local.variable_lookup, lookup(config, "configurations", ""), [])), [])
          ) : (
          # It's already a list - keep as is
          lookup(config, "configurations", [])
        )
      )
    })
  }

  # Step 0.5: Resolve variable references within individual configuration field values
  # This allows fields like blockedHosts or blockedPermissions to use variable references:
  #   configurations:
  #     - blockedHosts: "{default_blocked_hosts}"
  #       allowedHosts:
  #         - "https://example.com"
  extensions_with_resolved_config_fields = {
    for key, config in local.extensions_with_resolved_configs :
    key => merge(config, {
      configurations = [
        for cfg in config.configurations :
        {
          for field_name, field_value in cfg :
          field_name => (
            # If the field value is a string that matches a variable reference, resolve it
            try(tostring(field_value), null) != null ?
            lookup(local.variable_lookup, tostring(field_value), field_value) :
            field_value
          )
        }
      ]
    })
  }

  # Step 0.75: Expand simplified installationUrl into the new schema name.
  # chrome.users.apps.InstallationUrl was deprecated in Aug 2023 in favour of
  # chrome.users.apps.AppInstallationUrl. This allows YAML configs to continue
  # writing `installationUrl: <url>` and have the module emit the correct schema.
  # Note: OverrideInstallationUrl is NOT emitted because the Chrome Policy API
  # does not return it for most extensions — only emit it if explicitly configured.
  extensions_with_expanded_configs = {
    for key, config in local.extensions_with_resolved_config_fields :
    key => merge(config, {
      configurations = concat(
        # All configs that are not a simplified installationUrl — pass through unchanged
        [for cfg in config.configurations :
          cfg
          if !(lookup(cfg, "schema_name", null) == null && lookup(cfg, "installationUrl", null) != null)
        ],
        # Simplified installationUrl → AppInstallationUrl
        [for cfg in config.configurations :
          { schema_name = "chrome.users.apps.AppInstallationUrl", schema_values = { installationUrl = cfg.installationUrl } }
          if lookup(cfg, "schema_name", null) == null && lookup(cfg, "installationUrl", null) != null
        ]
      )
    })
  }

  # Step 1: Process configurations - transform simplified format to standard format
  extensions_with_processed_configs = {
    for key, config in local.extensions_with_expanded_configs :
    key => merge(config, {
      # Get raw configurations (already resolved from variables)
      raw_configurations = config.configurations

      # Transform simplified configurations to standard format
      # Each config item can be either:
      # 1. { schema_name: "...", schema_values: {...} }
      # 2. { fieldName: value } - gets converted to proper schema format
      # Simplified items may contain fields from multiple schemas; flatten into
      # one entry per schema so fields are not incorrectly grouped together.
      processed_configurations = flatten([
        for cfg in config.configurations : (
          # Check if this has schema_name
          lookup(cfg, "schema_name", null) != null ? [{
            # Standard format - pass through schema_name and schema_values
            schema_name   = cfg.schema_name
            schema_values = cfg.schema_values
          }] :
          # Simplified format - split by schema so each schema gets its own entry
          [
            for schema_name in distinct([
              for field_name, _ in cfg :
              local.field_to_schema_map[field_name]
              if lookup(local.field_to_schema_map, field_name, null) != null
              ]) : {
              schema_name = schema_name
              schema_values = {
                for field_name, field_value in cfg :
                field_name => field_value
                if lookup(local.field_to_schema_map, field_name, null) != null
                && local.field_to_schema_map[field_name] == schema_name
              }
            }
          ]
        )
      ])
    })
  }

  # Step 2: Group configurations by schema_name and merge schema values
  extensions_with_merged_configs = {
    for key, config in local.extensions_with_processed_configs :
    key => merge(config, {
      # Group configurations by schema_name to consolidate fields that belong to the same schema
      # e.g., blockedPermissions and allowedPermissions both go to PermissionsAndUrlAccess
      merged_configurations = [
        for schema_name in distinct([
          for cfg in config.processed_configurations :
          cfg.schema_name
        ]) :
        {
          schema_name = schema_name
          schema_values = merge([
            for cfg in config.processed_configurations :
            cfg.schema_values
            if cfg.schema_name == schema_name
          ]...)
        }
      ]
    })
  }

  # Step 3: Resolve target IDs and build final policies
  extensions_with_resolved_targets = {
    for key, config in local.extensions_with_merged_configs :
    key => merge(config, {
      # Resolve group ID if this is a group extension
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

      # Resolve OU ID if this is an OU extension
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

      # Build the policies list - includes InstallType if specified, plus any additional configurations
      all_policies = concat(
        # Add InstallType policy if install_type is specified
        lookup(config, "install_type", null) != null ? [{
          schema_name = "chrome.users.apps.InstallType"
          schema_values = {
            appInstallType = config.install_type
          }
        }] : [],
        # Add merged configurations
        config.merged_configurations
      )
    })
  }

  # Separate extensions by type
  group_extensions = {
    for key, config in local.extensions_with_resolved_targets :
    key => config
    if config.is_group_extension && !config.is_ou_extension
  }

  ou_extensions = {
    for key, config in local.extensions_with_resolved_targets :
    key => config
    if config.is_ou_extension && !config.is_group_extension
  }

  # ============================================================================
  # CONFLICT DETECTION: Detect duplicate extension+target combinations
  # ============================================================================

  # Build a map of all extension entries with their unique target identifier
  # Key: extension_id + target_type + target_id
  # This helps detect when the same extension is configured for the same target multiple times
  extension_target_entries = [
    for key, config in local.extensions_with_resolved_targets : {
      config_key   = key
      extension_id = config.full_app_id
      target_type  = config.is_group_extension ? "group" : (config.is_ou_extension ? "ou" : "unknown")
      target_id    = config.is_group_extension ? coalesce(config.resolved_group_id, "unresolved") : coalesce(config.resolved_ou_id, "unresolved")
      install_type = lookup(config, "install_type", "")
      source_file  = config._source_file
      from_group   = lookup(config, "_from_group", null)
      # Create a unique key for the extension+target combination
      unique_key = "${config.full_app_id}|${config.is_group_extension ? "group" : "ou"}|${config.is_group_extension ? coalesce(config.resolved_group_id, "unresolved") : coalesce(config.resolved_ou_id, "unresolved")}"
    }
    if config.full_app_id != null
  ]

  # Group entries by their unique extension+target key
  entries_by_unique_key = {
    for entry in local.extension_target_entries :
    entry.unique_key => entry...
  }

  # Find duplicates: entries where the same extension+target appears more than once
  duplicate_extension_targets = {
    for unique_key, entries in local.entries_by_unique_key :
    unique_key => entries
    if length(entries) > 1
  }

  # Format duplicate conflicts for error messages
  duplicate_extension_errors = [
    for unique_key, entries in local.duplicate_extension_targets :
    "Extension '${entries[0].extension_id}' for ${entries[0].target_type} '${entries[0].target_id}' is configured ${length(entries)} times: ${join(", ", [
      for e in entries :
      e.from_group != null ? "'${e.config_key}' (from extension_group '${e.from_group}' in ${e.source_file})" : "'${e.config_key}' (in ${e.source_file})"
    ])}. Install types: [${join(", ", distinct([for e in entries : e.install_type]))}]"
  ]

  # ============================================================================
  # STANDARD VALIDATION
  # ============================================================================

  # Validate that all extensions have a valid target reference
  invalid_group_extensions = [
    for key, config in local.group_extensions :
    key if config.resolved_group_id == null
  ]

  invalid_ou_extensions = [
    for key, config in local.ou_extensions :
    key if config.resolved_ou_id == null
  ]

  # Extensions that have both group and OU specified (invalid)
  conflicting_target_extensions = [
    for key, config in local.extensions_with_resolved_targets :
    key if config.is_group_extension && config.is_ou_extension
  ]

  # Extensions that have neither group nor OU specified (invalid)
  missing_target_extensions = [
    for key, config in local.extensions_with_resolved_targets :
    key if !config.is_group_extension && !config.is_ou_extension
  ]

  # Extensions that don't have a valid app_id
  missing_app_id_extensions = [
    for key, config in local.extensions_with_resolved_targets :
    key if config.full_app_id == null
  ]

  # Combine all validation errors
  has_validation_errors = (
    length(local.duplicate_extension_errors) > 0 ||
    length(local.invalid_group_extensions) > 0 ||
    length(local.invalid_ou_extensions) > 0 ||
    length(local.conflicting_target_extensions) > 0 ||
    length(local.missing_target_extensions) > 0 ||
    length(local.missing_app_id_extensions) > 0
  )

  # ============================================================================
  # IMPORT SUPPORT
  # ============================================================================
  # Compute import IDs for extensions with import: true in their YAML config.
  # Extensions always have additional_target_keys (app_id), so the import ID
  # format is: <target_id>/app_id=<full_app_id>/<schema1>,<schema2>,...

  ou_extensions_to_import = {
    for key, config in local.ou_extensions :
    key => {
      import_id = join("/", [
        config.resolved_ou_id,
        "app_id=${config.full_app_id}",
        join(",", [for p in config.all_policies : p.schema_name])
      ])
    }
    if lookup(config, "import", false) == true
  }

  group_extensions_to_import = {
    for key, config in local.group_extensions :
    key => {
      import_id = join("/", [
        config.resolved_group_id,
        "app_id=${config.full_app_id}",
        join(",", [for p in config.all_policies : p.schema_name])
      ])
    }
    if lookup(config, "import", false) == true
  }

}

# Validation using terraform_data with preconditions - fails at PLAN time
resource "terraform_data" "validate_no_duplicate_extensions" {
  lifecycle {
    precondition {
      condition     = length(local.duplicate_extension_errors) == 0
      error_message = <<-EOT
        DUPLICATE EXTENSION CONFLICTS DETECTED:
        The same extension cannot be configured multiple times for the same target (group or OU).

        ${join("\n\n        ", local.duplicate_extension_errors)}

        Resolution: Remove duplicate configurations or consolidate into a single entry.
      EOT
    }
  }
}

resource "terraform_data" "validate_group_references" {
  lifecycle {
    precondition {
      condition     = length(local.invalid_group_extensions) == 0
      error_message = <<-EOT
        INVALID GROUP REFERENCES:
        The following extensions reference groups that don't exist in groups_map:
        ${join(", ", local.invalid_group_extensions)}

        Resolution: Ensure the group_key exists in the groups module output or use group_email instead.
      EOT
    }
  }
}

resource "terraform_data" "validate_ou_references" {
  lifecycle {
    precondition {
      condition     = length(local.invalid_ou_extensions) == 0
      error_message = <<-EOT
        INVALID OU REFERENCES:
        The following extensions reference OUs that don't exist:
        ${join(", ", local.invalid_ou_extensions)}

        Resolution: Ensure the ou_key exists in the ou_map or use a valid ou_path.
      EOT
    }
  }
}

resource "terraform_data" "validate_single_target" {
  lifecycle {
    precondition {
      condition     = length(local.conflicting_target_extensions) == 0
      error_message = <<-EOT
        CONFLICTING TARGET SPECIFICATIONS:
        The following extensions have both group AND OU targets specified (only one allowed):
        ${join(", ", local.conflicting_target_extensions)}

        Resolution: Specify either a group target OR an OU target, not both.
      EOT
    }
  }
}

resource "terraform_data" "validate_has_target" {
  lifecycle {
    precondition {
      condition     = length(local.missing_target_extensions) == 0
      error_message = <<-EOT
        MISSING TARGET SPECIFICATIONS:
        The following extensions have no target (group or OU) specified:
        ${join(", ", local.missing_target_extensions)}

        Resolution: Add group_key, group_email, ou_key, or ou_path to each extension.
      EOT
    }
  }
}

resource "terraform_data" "validate_has_app_id" {
  lifecycle {
    precondition {
      condition     = length(local.missing_app_id_extensions) == 0
      error_message = <<-EOT
        MISSING APP ID:
        The following extensions have no app identifier specified:
        ${join(", ", local.missing_app_id_extensions)}

        Resolution: Add extension_id, android_app_id, or web_app_url to each extension.
      EOT
    }
  }
}

# Create Chrome group extension policies
resource "googleworkspace_chrome_group_policy" "group_extension" {
  for_each = local.group_extensions

  group_id = each.value.resolved_group_id

  # Create policy blocks for each policy in the configuration
  dynamic "policies" {
    for_each = each.value.all_policies
    content {
      schema_name = policies.value.schema_name
      schema_values = {
        for k, v in policies.value.schema_values :
        # Bool fields (e.g. allowAccessToKeys: true) are coerced to the string "true"/"false"
        # by HCL's type unification in the variable-lookup step. They are already valid JSON
        # booleans — applying jsonencode again would produce "\"true\"" (double-encoded).
        # Use can(tobool(v)) to detect bool-as-string; both branches must return string to
        # satisfy Terraform's static type checker (v can also be a tuple for list fields).
        k => can(tobool(v)) ? tostring(tobool(v)) : jsonencode(v)
      }
    }
  }

  # Add app_id as additional_target_keys
  additional_target_keys {
    target_key   = "app_id"
    target_value = each.value.full_app_id
  }

  depends_on = [
    terraform_data.validate_no_duplicate_extensions,
    terraform_data.validate_group_references,
    terraform_data.validate_ou_references,
    terraform_data.validate_single_target,
    terraform_data.validate_has_target,
    terraform_data.validate_has_app_id,
  ]
}

# Create Chrome OU extension policies
resource "googleworkspace_chrome_policy" "ou_extension" {
  for_each = local.ou_extensions

  org_unit_id = each.value.resolved_ou_id

  # Create policy blocks for each policy in the configuration
  dynamic "policies" {
    for_each = each.value.all_policies
    content {
      schema_name = policies.value.schema_name
      schema_values = {
        for k, v in policies.value.schema_values :
        k => can(tobool(v)) ? tostring(tobool(v)) : jsonencode(v)
      }
    }
  }

  # Add app_id as additional_target_keys
  additional_target_keys {
    target_key   = "app_id"
    target_value = each.value.full_app_id
  }

  depends_on = [
    terraform_data.validate_no_duplicate_extensions,
    terraform_data.validate_group_references,
    terraform_data.validate_ou_references,
    terraform_data.validate_single_target,
    terraform_data.validate_has_target,
    terraform_data.validate_has_app_id,
    googleworkspace_chrome_group_policy.group_extension,
  ]
}
