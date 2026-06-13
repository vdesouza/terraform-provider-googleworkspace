variable "yaml_config_path" {
  description = "Path to the directory containing YAML group priority ordering configuration files"
  type        = string
}

variable "groups_map" {
  description = "Map of group emails to group details (from groups module output or data sources)"
  type = map(object({
    id    = string
    email = string
  }))
}

variable "policies_map" {
  description = "Map of policy keys to policy details (from policies module output) for determining which groups have which policies"
  type = map(object({
    group_id    = string
    group_email = string
    policies = list(object({
      schema_name = string
    }))
  }))
}

variable "external_policy_orderings" {
  description = "Map of policy-specific orderings from external sources (e.g., policy YAML files). These are merged with orderings defined in the group_priority YAML files."
  type        = map(list(string))
  default     = {}
}

variable "external_policy_orderings_with_source" {
  description = "Map of policy orderings with source file information from external sources (e.g., policy YAML files) for duplicate validation"
  type = map(object({
    source_file = string
    ordering    = list(string)
  }))
  default = {}
}

# Extension ordering variables (from extensions module)
variable "extensions_map" {
  description = "Map of extension keys to extension details (from extensions module output) for determining which groups have which extensions"
  type = map(object({
    group_id    = string
    group_email = string
    app_id      = string
    policies = list(object({
      schema_name = string
    }))
  }))
  default = {}
}

variable "external_extension_orderings" {
  description = "Map of extension-specific orderings from external sources (e.g., extension YAML files). These are merged with orderings defined in the group_priority YAML files."
  type        = map(list(string))
  default     = {}
}

variable "external_extension_orderings_with_source" {
  description = "Map of extension orderings with source file information from external sources (e.g., extension YAML files) for duplicate validation"
  type = map(object({
    source_file = string
    ordering    = list(string)
  }))
  default = {}
}

# Note: default_extension_ordering must be defined in group_priority YAML files only
# (similar to default_policy_ordering). Only extension_orderings can come from external sources.

variable "variables_map" {
  description = "Map of variable names to values (from variables module) for resolving {varname} references in YAML configs"
  type        = any
  default     = {}
}
