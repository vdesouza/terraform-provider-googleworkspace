variable "yaml_config_path" {
  description = "Path to the directory containing YAML group configuration files"
  type        = string
}

variable "variables_map" {
  description = "Map of variable names to values (from variables module) for resolving {varname} references in YAML configs"
  type        = any
  default     = {}
}
