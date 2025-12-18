data "googleworkspace_org_unit" "example" {
  org_unit_path = "/"
}

# Retrieve the group priority ordering for a Chrome policy
data "googleworkspace_chrome_policy_group_priority_ordering" "example" {
  policy_schema = "chrome.users.MaxConnectionsPerProxy"

  policy_target_key {
    target_resource = data.googleworkspace_org_unit.example.id

    # The target app must be supplied in additional_target_key_names
    additional_target_key_names = {
      app_id = "chrome"
    }
  }
}

# The group_ids attribute contains the ordered list of group IDs
# Groups earlier in the list have higher priority
output "group_priority_order" {
  value = data.googleworkspace_chrome_policy_group_priority_ordering.example.group_ids
}
