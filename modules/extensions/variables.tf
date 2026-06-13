variable "yaml_config_paths" {
  description = "List of paths to directories containing YAML configuration files. Extensions can be defined in any YAML file with an 'extensions' section."
  type        = list(string)
}

variable "groups_map" {
  description = "Map of group keys to group details (from groups module output) for resolving group_key references"
  type = map(object({
    id           = string
    email        = string
    display_name = string
  }))
  default = {}
}

variable "ou_map" {
  description = "Map of OU keys to OU details for resolving ou_key references (optional, alternative to ou_path)"
  type = map(object({
    id   = string
    path = string
    name = string
  }))
  default = {}
}

variable "variables_map" {
  description = "Map of variable names to values (from variables module) for resolving {varname} references in YAML configs"
  type        = any
  default     = {}
}
