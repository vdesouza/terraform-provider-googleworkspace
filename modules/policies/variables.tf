variable "yaml_config_paths" {
  description = "List of paths to directories containing YAML configuration files. Policies can be defined in any YAML file with a 'policies' section."
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

variable "assets_map" {
  description = "Map of asset keys to asset details (from assets module output) for resolving asset references in policies. Key format: asset_key/policy_field"
  type = map(object({
    asset_key    = string
    policy_field = string
    file         = string
    download_uri = string
    description  = string
    comments     = string
    content_type = string
    file_hash    = string
  }))
  default = {}
}

variable "variables_map" {
  description = "Map of variable names to values (from variables module) for resolving {varname} references in YAML configs"
  type        = any
  default     = {}
}
