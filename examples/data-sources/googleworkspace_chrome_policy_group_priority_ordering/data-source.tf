# Example 1: Retrieve group priority ordering for chrome.users.* policies
# (does not require policy_target_key)

data "googleworkspace_chrome_policy_group_priority_ordering" "wallpaper" {
  policy_schema = "chrome.users.Wallpaper"
}

# The group_ids attribute contains the ordered list of group IDs
# Groups earlier in the list have higher priority
output "wallpaper_group_priority_order" {
  value = data.googleworkspace_chrome_policy_group_priority_ordering.wallpaper.group_ids
}

# Example 2: Retrieve group priority ordering for chrome.apps.* policies
# (requires policy_target_key)

data "googleworkspace_org_unit" "sales" {
  org_unit_path = "/Sales"
}

data "googleworkspace_chrome_policy_group_priority_ordering" "app_specific" {
  policy_schema = "chrome.apps.ExampleAppPolicy"

  # For chrome.apps.* policies, policy_target_key is required
  policy_target_key {
    target_resource = data.googleworkspace_org_unit.sales.id

    # The target app must be supplied in additional_target_key_names
    additional_target_key_names = {
      app_id = "chrome:example_app_id"
    }
  }
}

output "app_specific_group_priority_order" {
  value = data.googleworkspace_chrome_policy_group_priority_ordering.app_specific.group_ids
}
