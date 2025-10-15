package googleworkspace

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceGroupDynamic_basic(t *testing.T) {
	t.Parallel()

	domainName := os.Getenv("GOOGLEWORKSPACE_DOMAIN")

	if domainName == "" {
		t.Skip("GOOGLEWORKSPACE_DOMAIN needs to be set to run this test")
	}

	testGroupVals := map[string]interface{}{
		"domainName": domainName,
		"email":      fmt.Sprintf("tf-test-dynamic-%s", acctest.RandString(10)),
	}

	expectedEmail := fmt.Sprintf("%s@%s", testGroupVals["email"].(string), testGroupVals["domainName"].(string))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceGroupDynamic_basic(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "email", expectedEmail),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "query", "user.organizations.exists(org, org.department == 'Engineering')"),
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "name"),
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "group_key_id"),
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "create_time"),
				),
			},
			{
				// TestStep imports by resource name (format: groups/{group_id})
				ResourceName:            "googleworkspace_group_dynamic.test-dynamic",
				ImportState:             true,
				ImportStateCheck:        checkGroupDynamicImportState(),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
		},
	})
}

func checkGroupDynamicImportState() resource.ImportStateCheckFunc {
	return resource.ImportStateCheckFunc(
		func(state []*terraform.InstanceState) error {
			if len(state) > 1 {
				return fmt.Errorf("state should only contain one dynamic group resource, got: %d", len(state))
			}

			id := state[0].ID
			// ID should be in format "groups/{group_id}"
			if len(id) < 7 || id[:7] != "groups/" {
				return fmt.Errorf("id should be in format 'groups/{group_id}', got: %s", id)
			}

			return nil
		},
	)
}

func TestAccResourceGroupDynamic_full(t *testing.T) {
	t.Parallel()

	domainName := os.Getenv("GOOGLEWORKSPACE_DOMAIN")

	if domainName == "" {
		t.Skip("GOOGLEWORKSPACE_DOMAIN needs to be set to run this test")
	}

	testGroupVals := map[string]interface{}{
		"domainName": domainName,
		"email":      fmt.Sprintf("tf-test-dynamic-%s", acctest.RandString(10)),
	}

	expectedEmail := fmt.Sprintf("%s@%s", testGroupVals["email"].(string), testGroupVals["domainName"].(string))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceGroupDynamic_full(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "email", expectedEmail),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "display_name", "Test Dynamic Group"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "description", "A test dynamic group"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "query", "user.organizations.exists(org, org.department == 'Engineering')"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "security_group", "true"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "locked", "false"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "labels.team", ""),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "labels.environment", ""),
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "name"),
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "create_time"),
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "update_time"),
				),
			},
			{
				ResourceName:            "googleworkspace_group_dynamic.test-dynamic",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			{
				// Update the group
				Config: testAccResourceGroupDynamic_fullUpdate(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "display_name", "Updated Dynamic Group"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "description", "An updated test dynamic group"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "query", "user.organizations.exists(org, org.department == 'Engineering' || org.department == 'IT')"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "security_group", "true"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "locked", "true"),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "labels.team", ""),
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "labels.updated", ""),
				),
			},
			{
				ResourceName:            "googleworkspace_group_dynamic.test-dynamic",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
		},
	})
}

func TestAccResourceGroupDynamic_queryValidation(t *testing.T) {
	t.Parallel()

	domainName := os.Getenv("GOOGLEWORKSPACE_DOMAIN")

	if domainName == "" {
		t.Skip("GOOGLEWORKSPACE_DOMAIN needs to be set to run this test")
	}

	testGroupVals := map[string]interface{}{
		"domainName": domainName,
		"email":      fmt.Sprintf("tf-test-dynamic-%s", acctest.RandString(10)),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Test with email-based query
				Config: testAccResourceGroupDynamic_emailQuery(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "query", "user.email.contains('@contractor.example.com')"),
				),
			},
			{
				// Test with location-based query
				Config: testAccResourceGroupDynamic_locationQuery(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "query", "user.locations.exists(loc, loc.buildingId == 'NYC-01')"),
				),
			},
			{
				// Test with complex multi-condition query
				Config: testAccResourceGroupDynamic_complexQuery(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("googleworkspace_group_dynamic.test-dynamic", "query"),
				),
			},
		},
	})
}

func TestAccResourceGroupDynamic_minimalConfig(t *testing.T) {
	t.Parallel()

	domainName := os.Getenv("GOOGLEWORKSPACE_DOMAIN")

	if domainName == "" {
		t.Skip("GOOGLEWORKSPACE_DOMAIN needs to be set to run this test")
	}

	testGroupVals := map[string]interface{}{
		"domainName": domainName,
		"email":      fmt.Sprintf("tf-test-dynamic-%s", acctest.RandString(10)),
	}

	expectedEmail := fmt.Sprintf("%s@%s", testGroupVals["email"].(string), testGroupVals["domainName"].(string))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceGroupDynamic_basic(testGroupVals),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "email", expectedEmail),
					// display_name should default to email
					resource.TestCheckResourceAttr("googleworkspace_group_dynamic.test-dynamic", "display_name", expectedEmail),
				),
			},
		},
	})
}

// Test configuration templates

func testAccResourceGroupDynamic_basic(testGroupVals map[string]interface{}) string {
	return Nprintf(`
resource "googleworkspace_group_dynamic" "test-dynamic" {
  email = "%{email}@%{domainName}"
  query = "user.organizations.exists(org, org.department == 'Engineering')"
}
`, testGroupVals)
}

func testAccResourceGroupDynamic_full(testGroupVals map[string]interface{}) string {
	return Nprintf(`
resource "googleworkspace_group_dynamic" "test-dynamic" {
  email          = "%{email}@%{domainName}"
  display_name   = "Test Dynamic Group"
  description    = "A test dynamic group"
  query          = "user.organizations.exists(org, org.department == 'Engineering')"
  security_group = true
  locked         = false
  
  labels = {
    team        = ""
    environment = ""
  }
}
`, testGroupVals)
}

func testAccResourceGroupDynamic_fullUpdate(testGroupVals map[string]interface{}) string {
	return Nprintf(`
resource "googleworkspace_group_dynamic" "test-dynamic" {
  email          = "%{email}@%{domainName}"
  display_name   = "Updated Dynamic Group"
  description    = "An updated test dynamic group"
  query          = "user.organizations.exists(org, org.department == 'Engineering' || org.department == 'IT')"
  security_group = true
  locked         = true
  
  labels = {
    team    = ""
    updated = ""
  }
}
`, testGroupVals)
}

func testAccResourceGroupDynamic_emailQuery(testGroupVals map[string]interface{}) string {
	return Nprintf(`
resource "googleworkspace_group_dynamic" "test-dynamic" {
  email = "%{email}@%{domainName}"
  query = "user.email.contains('@contractor.example.com')"
}
`, testGroupVals)
}

func testAccResourceGroupDynamic_locationQuery(testGroupVals map[string]interface{}) string {
	return Nprintf(`
resource "googleworkspace_group_dynamic" "test-dynamic" {
  email = "%{email}@%{domainName}"
  query = "user.locations.exists(loc, loc.buildingId == 'NYC-01')"
}
`, testGroupVals)
}

func testAccResourceGroupDynamic_complexQuery(testGroupVals map[string]interface{}) string {
	return Nprintf(`
resource "googleworkspace_group_dynamic" "test-dynamic" {
  email = "%{email}@%{domainName}"
  query = "user.organizations.exists(org, org.department == 'Engineering' && (org.title.contains('Senior') || org.title.contains('Lead'))) && user.addresses.exists(addr, addr.country == 'US')"
}
`, testGroupVals)
}
