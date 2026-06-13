# Variables Module
# Parses YAML files from config/variables/ to provide reusable variables
# that can be referenced in other YAML configs using {variable_name} syntax.

locals {
  # Get all YAML files recursively from the variables directory
  yaml_files = fileset(var.yaml_config_path, "**/*.{yml,yaml}")

  # Parse each YAML file
  parsed_yaml_files = {
    for file in local.yaml_files :
    file => try(yamldecode(file("${var.yaml_config_path}/${file}")), {})
  }

  # Track which file each variable comes from (for duplicate detection)
  variables_by_file = {
    for file, content in local.parsed_yaml_files :
    file => {
      for var_name, var_value in content :
      var_name => {
        value       = var_value
        source_file = file
      }
    }
  }

  # Flatten to get all variable names with their sources
  all_variables_with_source = merge([
    for file, vars in local.variables_by_file :
    vars
  ]...)

  # Detect duplicate variable names across files
  variable_sources = {
    for var_name in distinct(flatten([
      for file, vars in local.variables_by_file :
      keys(vars)
    ])) :
    var_name => [
      for file, vars in local.variables_by_file :
      file
      if contains(keys(vars), var_name)
    ]
  }

  # Find variables defined in multiple files
  duplicate_variables = {
    for var_name, sources in local.variable_sources :
    var_name => sources
    if length(sources) > 1
  }

  # Validation errors for duplicates
  duplicate_variable_errors = [
    for var_name, sources in local.duplicate_variables :
    "Variable '${var_name}' is defined in multiple files: ${join(", ", sources)}. Each variable must be defined only once."
  ]

  # Flatten all variables from all files into single map (values only)
  # Later files override earlier ones if there are duplicates (but validation catches this)
  variables_map = merge([
    for file, content in local.parsed_yaml_files :
    content
  ]...)

  # Build the lookup map for variable resolution
  # Key: "{varname}", Value: variable value (string or list)
  variable_lookup = {
    for var_name, var_value in local.variables_map :
    "{${var_name}}" => var_value
  }
}

# Validation resource - fails during plan if duplicate variables exist
resource "terraform_data" "validate_no_duplicate_variables" {
  lifecycle {
    precondition {
      condition     = length(local.duplicate_variable_errors) == 0
      error_message = <<-EOT
        DUPLICATE VARIABLE DEFINITIONS DETECTED:
        ${join("\n", local.duplicate_variable_errors)}

        Resolution: Ensure each variable name is defined in only one file.
      EOT
    }
  }
}
