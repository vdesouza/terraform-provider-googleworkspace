output "static_groups" {
  description = "Map of static groups created"
  value = {
    for key, group in googleworkspace_group.static :
    key => {
      id    = group.id
      email = group.email
      name  = group.name
    }
  }
}

output "dynamic_groups" {
  description = "Map of dynamic group keys to their details"
  value = {
    for key, group in googleworkspace_group_dynamic.dynamic : key => {
      id           = split("/", group.name)[1] # Extract group_id from groups/{group_id}
      name         = group.name                # Full resource name: groups/{group_id}
      email        = group.email
      display_name = group.display_name
      query        = group.query
    }
  }
}

output "all_groups" {
  description = "Map of all group keys to their details with type indicator"
  value = merge(
    {
      for key, group in googleworkspace_group.static : key => {
        id           = group.id
        email        = local.static_groups[key].email # Use configured email — always known at plan time
        display_name = group.name
        type         = "static"
      }
    },
    {
      for key, group in googleworkspace_group_dynamic.dynamic : key => {
        id           = split("/", group.name)[1]       # Extract group_id from groups/{group_id}
        name         = group.name                      # Full resource name
        email        = local.dynamic_groups[key].email # Use configured email — always known at plan time
        display_name = group.display_name
        type         = "dynamic"
        query        = group.query
      }
    }
  )
}
