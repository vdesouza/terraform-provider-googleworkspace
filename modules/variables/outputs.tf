output "variables_map" {
  description = "Map of all variables defined in YAML files (var_name => value)"
  value       = local.variables_map
}

output "variable_lookup" {
  description = "Lookup map for variable resolution ({varname} => value)"
  value       = local.variable_lookup
}

output "variable_names" {
  description = "List of all defined variable names"
  value       = keys(local.variables_map)
}

output "variables_by_file" {
  description = "Variables grouped by source file (for debugging)"
  value = {
    for file, vars in local.variables_by_file :
    file => keys(vars)
  }
}
