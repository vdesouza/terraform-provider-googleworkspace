# Groups Module

This module manages Google Workspace groups from YAML configuration files. It supports both static groups (with explicit members) and dynamic groups (with queries).

## Features

- Manage multiple groups from YAML files
- Support for multiple YAML files in a single directory
- Automatic detection of static vs dynamic groups
- Build dynamic group queries from org unit paths
- Combine multiple org units with OR logic
- Support for custom queries
- Automatic user.suspended filtering

## Usage

### Module Invocation

```hcl
module "groups" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/groups?ref=v1.4.0"
  
  yaml_config_path = "${path.module}/config/groups"
}
```

## YAML Schema

See [YAML_SCHEMA.md](YAML_SCHEMA.md) for full schema details.

## Member Roles

The module supports three roles for both users and groups:

- **MEMBER**: Can subscribe to the group, view discussions, and view membership list
- **MANAGER**: Can do everything an OWNER can except make members OWNERS or delete the group (requires Google Groups for Business)
- **OWNER**: Can send messages, add/remove members, change roles, modify settings, and delete the group

## Group Type Detection

The module automatically determines whether to create a static or dynamic group:

- **Dynamic Group**: Created when `query` or `org_units` is provided
- **Static Group**: Created when neither `query` nor `org_units` is provided

## Org Unit Query Building

When using `org_units`, the module:

1. Fetches org unit IDs using the `googleworkspace_org_unit` data source
2. Builds a query for each org unit: `user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('<id>'))`
3. Combines all org units with OR logic: `(query1) || (query2) || (query3)`
4. If both `query` and `org_units` are provided, they are ANDed together: `(custom_query) && ((ou_query1) || (ou_query2))`
5. Adds suspended filter if enabled: `((query)) && user.suspended == false`

This creates a query that matches users who are **direct or indirect members** of any of the specified org units.

## Multiple YAML Files

You can split your groups across multiple YAML files:

``` text
config/groups/
├── engineering_groups.yaml
├── partner_groups.yaml
└── admin_groups.yaml
```

All files will be read and merged together. The top-level key for each group must be unique across all files.

## Outputs

The module provides the following outputs:

- `static_groups`: Map of all static groups created (includes id, email, name)
- `dynamic_groups`: Map of all dynamic groups created (includes id, email, display_name, query)
- `all_groups`: Combined map of all groups with type information

### Using Outputs in Other Modules

```hcl
module "groups" {
  source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/groups?ref=v1.4.0"
  yaml_config_path = "${path.module}/config/groups"
}

# Reference group by key
resource "some_resource" "example" {
  group_id = module.groups.static_groups["my_static_group"].id
  group_email = module.groups.all_groups["my_dynamic_group"].email
}
```

## Examples

See `config/groups/example_groups.yaml` for comprehensive examples.

## References

- [Google Workspace Dynamic Groups Query Language](https://cloud.google.com/identity/docs/how-to/test-query-dynamic-groups)
- [Terraform Provider Documentation](https://github.com/vdesouza/terraform-provider-googleworkspace)
