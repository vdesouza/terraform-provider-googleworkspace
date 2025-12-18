resource "googleworkspace_org_unit" "example" {
  name                 = "example"
  parent_org_unit_path = "/"
}

resource "googleworkspace_group" "priority_group_1" {
  email = "high-priority@example.com"
  name  = "High Priority Group"
}

resource "googleworkspace_group" "priority_group_2" {
  email = "medium-priority@example.com"
  name  = "Medium Priority Group"
}

resource "googleworkspace_group" "priority_group_3" {
  email = "low-priority@example.com"
  name  = "Low Priority Group"
}

# Set group priority ordering for a Chrome policy
# Groups earlier in the list have higher priority and their policies will override those of groups later in the list
resource "googleworkspace_chrome_policy_group_priority_ordering" "example" {
  policy_schema = "chrome.users.MaxConnectionsPerProxy"

  policy_target_key {
    target_resource = googleworkspace_org_unit.example.id

    # The target app must be supplied in additional_target_key_names
    additional_target_key_names = {
      app_id = "chrome"
    }
  }

  # Groups in priority order (highest to lowest)
  # Policies from high-priority group will override those from medium and low priority groups
  group_ids = [
    googleworkspace_group.priority_group_1.id,
    googleworkspace_group.priority_group_2.id,
    googleworkspace_group.priority_group_3.id,
  ]
}
