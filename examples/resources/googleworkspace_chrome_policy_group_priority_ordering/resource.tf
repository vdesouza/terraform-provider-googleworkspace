# Example 1: Group priority ordering for chrome.users.* policies
# (does not require policy_target_key)

resource "googleworkspace_group" "high_priority" {
  email = "high-priority@example.com"
  name  = "High Priority Group"
}

resource "googleworkspace_group" "medium_priority" {
  email = "medium-priority@example.com"
  name  = "Medium Priority Group"
}

resource "googleworkspace_group" "low_priority" {
  email = "low-priority@example.com"
  name  = "Low Priority Group"
}

# For chrome.users.* policies, policy_target_key is not required
resource "googleworkspace_chrome_policy_group_priority_ordering" "wallpaper" {
  policy_schema = "chrome.users.Wallpaper"

  # Groups in priority order (highest to lowest)
  # Policies from high-priority group will override those from medium and low priority groups
  group_ids = [
    googleworkspace_group.high_priority.id,
    googleworkspace_group.medium_priority.id,
    googleworkspace_group.low_priority.id,
  ]
}

# Example 2: Group priority ordering for chrome.apps.* policies
# (requires policy_target_key with target app)

resource "googleworkspace_org_unit" "sales" {
  name                 = "Sales"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy_group_priority_ordering" "app_specific" {
  policy_schema = "chrome.apps.ExampleAppPolicy"

  # For chrome.apps.* policies, policy_target_key is required
  policy_target_key {
    target_resource = googleworkspace_org_unit.sales.id

    # The target app must be supplied in additional_target_key_names
    additional_target_key_names = {
      app_id = "chrome:example_app_id"
    }
  }

  # Groups in priority order (highest to lowest)
  group_ids = [
    googleworkspace_group.high_priority.id,
    googleworkspace_group.medium_priority.id,
  ]
}
