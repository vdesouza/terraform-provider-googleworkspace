resource "googleworkspace_group_dynamic" "engineering" {
  email        = "dynamic-engineering@example.com"
  display_name = "Engineering Team (Dynamic)"
  description  = "All members of the Engineering department"

  # Dynamic query to automatically add users from Engineering department
  query = "user.organizations.exists(org, org.department == 'Engineering')"

  # Make this a security group (immutable - cannot be removed once added)
  security_label = true

  # Custom labels - all values must be empty strings
  labels = {
    "team" = ""
  }
}

# Example: Dynamic group based on email domain
resource "googleworkspace_group_dynamic" "contractors" {
  email        = "contractors@example.com"
  display_name = "All Contractors"

  query = "user.email.contains('@contractor.example.com')"

  # Lock the group to prevent manual member changes
  locked = true
}

# Example: Dynamic group with multiple conditions
resource "googleworkspace_group_dynamic" "senior_engineers" {
  email        = "senior-engineers@example.com"
  display_name = "Senior Engineers"
  description  = "Senior engineering staff in the US"

  # Query with AND conditions
  query = "user.organizations.exists(org, org.department == 'Engineering' && org.title.contains('Senior')) && user.addresses.exists(addr, addr.country == 'US')"

  # Security group that's locked
  security_label = true
  locked         = true

  labels = {
    "department"   = ""
    "access_level" = ""
  }
}

# Example: Dynamic group based on custom schema
resource "googleworkspace_group_dynamic" "beta_testers" {
  email        = "beta-testers@example.com"
  display_name = "Beta Testers"

  # Assuming you have a custom schema field
  query = "user.customSchemas.exists(s, s.key == 'beta_program' && s.value == 'enrolled')"
}

# Example: Dynamic group for specific office locations
resource "googleworkspace_group_dynamic" "nyc_office" {
  email = "nyc-office@example.com"

  query = "user.locations.exists(loc, loc.buildingId == 'NYC-01' || loc.buildingId == 'NYC-02')"
}
