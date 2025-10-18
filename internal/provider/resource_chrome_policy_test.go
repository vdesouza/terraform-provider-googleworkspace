package googleworkspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"google.golang.org/api/chromepolicy/v1"
)

func TestAccResourceChromePolicy_basic(t *testing.T) {
	t.Parallel()

	ouName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicy_basic(ouName, 33),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.MaxConnectionsPerProxy"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.maxConnectionsPerProxy", "33"),
				),
			},
		},
	})
}

func TestAccResourceChromePolicy_typeMessage(t *testing.T) {
	t.Parallel()

	ouName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicy_typeMessage(ouName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.ManagedBookmarksSetting"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.managedBookmarks", "{\"toplevelName\":\"Stuff\"}"),
				),
			},
		},
	})
}

func TestAccResourceChromePolicy_additionalTargetKey(t *testing.T) {
	t.Parallel()

	ouName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicy_additionalTargetKey(ouName, "chrome:glnpjglilkicbckjpbgcfkogebgllemb", "ALLOWED"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.apps.InstallType"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.appInstallType", encode("ALLOWED")),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "additional_target_keys.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "additional_target_keys.0.target_key", "app_id"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "additional_target_keys.0.target_value", "chrome:glnpjglilkicbckjpbgcfkogebgllemb"),
				),
			},
		},
	})
}

func TestAccResourceChromePolicy_update(t *testing.T) {
	t.Parallel()

	ouName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicy_basic(ouName, 33),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.MaxConnectionsPerProxy"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.maxConnectionsPerProxy", "33"),
				),
			},
			{
				Config: testAccResourceChromePolicy_basic(ouName, 34),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.MaxConnectionsPerProxy"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.maxConnectionsPerProxy", "34"),
				),
			},
		},
	})
}

func TestAccResourceChromePolicy_multiple(t *testing.T) {
	t.Parallel()

	ouName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	// ensures previously set field was reset/removed
	// this passing also implies Delete works correctly
	// based on the implementation
	testCheck := func(s *terraform.State) error {
		client, err := googleworkspaceTestClient()
		if err != nil {
			return err
		}

		rs, ok := s.RootModule().Resources["googleworkspace_org_unit.test"]
		if !ok {
			return fmt.Errorf("Can't find org unit resource: googleworkspace_org_unit.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("org unit ID not set")
		}

		chromePolicyService, diags := client.NewChromePolicyService()
		if diags.HasError() {
			return errors.New(diags[0].Summary)
		}

		chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
		if diags.HasError() {
			return errors.New(diags[0].Summary)
		}

		policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
			TargetResource: "orgunits/" + strings.TrimPrefix(rs.Primary.ID, "id:"),
		}

		resp, err := chromePoliciesService.Resolve(fmt.Sprintf("customers/%s", client.Customer), &chromepolicy.GoogleChromePolicyVersionsV1ResolveRequest{
			PolicySchemaFilter: "chrome.users.MaxConnectionsPerProxy",
			PolicyTargetKey:    policyTargetKey,
		}).Do()
		if err != nil {
			return err
		}
		if len(resp.ResolvedPolicies) > 0 {
			return fmt.Errorf("Expected policy to be reset")
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicy_multiple(ouName, 33, ".*@example"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "2"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.RestrictSigninToPattern"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.restrictSigninToPattern", encode(".*@example")),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.1.schema_name", "chrome.users.MaxConnectionsPerProxy"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.1.schema_values.maxConnectionsPerProxy", "33"),
				),
			},
			{
				Config: testAccResourceChromePolicy_multipleRearranged(ouName, 34, ".*@example.com"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "2"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.MaxConnectionsPerProxy"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.maxConnectionsPerProxy", "34"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.1.schema_name", "chrome.users.RestrictSigninToPattern"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.1.schema_values.restrictSigninToPattern", encode(".*@example.com")),
				),
			},
			{
				Config: testAccResourceChromePolicy_multipleDifferent(ouName, true, ".*@example.com"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "2"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.OnlineRevocationChecks"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.enableOnlineRevocationChecks", "true"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.1.schema_name", "chrome.users.RestrictSigninToPattern"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.1.schema_values.restrictSigninToPattern", encode(".*@example.com")),
					testCheck,
				),
			},
			{
				Config: testAccResourceChromePolicy_multipleValueTypes(ouName, true, "POLICY_MODE_RECOMMENDED"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.#", "1"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_name", "chrome.users.DomainReliabilityAllowed"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.domainReliabilityAllowed", "true"),
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy.test", "policies.0.schema_values.domainReliabilityAllowedSettingGroupPolicyMode", encode("POLICY_MODE_RECOMMENDED")),
					testCheck,
				),
			},
		},
	})
}

func encode(content string) string {
	res, _ := json.Marshal(content)
	return string(res)
}

func testAccResourceChromePolicy_multiple(ouName string, conns int, pattern string) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
  org_unit_id = googleworkspace_org_unit.test.id
  policies {
    schema_name = "chrome.users.RestrictSigninToPattern"
    schema_values = {
      restrictSigninToPattern = jsonencode("%s")
    }
  }
  policies {
    schema_name = "chrome.users.MaxConnectionsPerProxy"
    schema_values = {
      maxConnectionsPerProxy = jsonencode(%d)
    }
  }
}
`, ouName, pattern, conns)
}

func testAccResourceChromePolicy_multipleRearranged(ouName string, conns int, pattern string) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
  org_unit_id = googleworkspace_org_unit.test.id
  policies {
    schema_name = "chrome.users.MaxConnectionsPerProxy"
    schema_values = {
      maxConnectionsPerProxy = jsonencode(%d)
    }
  }
  policies {
    schema_name = "chrome.users.RestrictSigninToPattern"
    schema_values = {
      restrictSigninToPattern = jsonencode("%s")
    }
  }
}
`, ouName, conns, pattern)
}

func testAccResourceChromePolicy_multipleDifferent(ouName string, enabled bool, pattern string) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
  org_unit_id = googleworkspace_org_unit.test.id
  policies {
    schema_name = "chrome.users.OnlineRevocationChecks"
    schema_values = {
      enableOnlineRevocationChecks = jsonencode(%t)
    }
  }
  policies {
    schema_name = "chrome.users.RestrictSigninToPattern"
    schema_values = {
      restrictSigninToPattern = jsonencode("%s")
    }
  }
}
`, ouName, enabled, pattern)
}

func testAccResourceChromePolicy_multipleValueTypes(ouName string, enabled bool, policyMode string) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
  org_unit_id = googleworkspace_org_unit.test.id
  policies {
    schema_name = "chrome.users.DomainReliabilityAllowed"
    schema_values = {
	  domainReliabilityAllowed                       = jsonencode(%t)
      domainReliabilityAllowedSettingGroupPolicyMode = jsonencode("%s")
    }
  }
}
`, ouName, enabled, policyMode)
}

func testAccResourceChromePolicy_basic(ouName string, conns int) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
  org_unit_id = googleworkspace_org_unit.test.id
  policies {
    schema_name = "chrome.users.MaxConnectionsPerProxy"
    schema_values = {
      maxConnectionsPerProxy = jsonencode(%d)
    }
  }
}
`, ouName, conns)
}

func testAccResourceChromePolicy_typeMessage(ouName string) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
  org_unit_id = googleworkspace_org_unit.test.id
  policies {
    schema_name = "chrome.users.ManagedBookmarksSetting"
    schema_values = {
		managedBookmarks = "{\"toplevelName\":\"Stuff\"}"
    }
  }
}
`, ouName)
}

func testAccResourceChromePolicy_additionalTargetKey(ouName string, app_id string, install_type string) string {
	return fmt.Sprintf(`
resource "googleworkspace_org_unit" "test" {
  name = "%s"
  parent_org_unit_path = "/"
}

resource "googleworkspace_chrome_policy" "test" {
org_unit_id = googleworkspace_org_unit.test.id
  additional_target_keys {
    target_key = "app_id"
	  target_value = "%s"
	}
  policies {
    schema_name = "chrome.users.apps.InstallType"
    schema_values = {
	  appInstallType = jsonencode("%s")
    }
  }
}
`, ouName, app_id, install_type)
}

// Unit tests for helper functions

func TestValidatePolicyFieldValueType(t *testing.T) {
	cases := []struct {
		fieldType string
		value     interface{}
		expect    bool
	}{
		{"TYPE_BOOL", true, true},
		{"TYPE_BOOL", "true", false},
		{"TYPE_DOUBLE", 1.23, true},
		{"TYPE_INT64", float64(10), true},
		{"TYPE_INT64", float64(10.5), false},
		{"TYPE_STRING", "abc", true},
		{"TYPE_ENUM", "SOME_ENUM", true},
		{"TYPE_MESSAGE", map[string]interface{}{"k": "v"}, true},
		{"TYPE_MESSAGE", []string{"x"}, false},
		{"TYPE_UINT32", float32(3), true},
		{"TYPE_UINT32", float32(3.1), false},
	}
	for _, c := range cases {
		if got := validatePolicyFieldValueType(c.fieldType, c.value); got != c.expect {
			t.Errorf("validatePolicyFieldValueType(%s,%v) expected %v got %v", c.fieldType, c.value, c.expect, got)
		}
	}
}

func TestConvertPolicyFieldValueType(t *testing.T) {
	cases := []struct {
		fieldType string
		in        interface{}
		want      interface{}
		wantErr   bool
	}{
		{"TYPE_BOOL", "true", true, false},
		{"TYPE_BOOL", "notbool", nil, true},
		{"TYPE_DOUBLE", "1.25", float64(1.25), false},
		{"TYPE_INT64", "42", int64(42), false},
		{"TYPE_INT64", "4.2", nil, true},
		{"TYPE_UINT32", "7", int64(7), false},
		{"TYPE_STRING", "abc", "abc", false},
		{"TYPE_ENUM", "ENUM_VAL", "ENUM_VAL", false},
	}
	for _, c := range cases {
		got, err := convertPolicyFieldValueType(c.fieldType, c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("expected error for %s input %v", c.fieldType, c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for %s input %v: %v", c.fieldType, c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("convertPolicyFieldValueType(%s,%v) expected %v got %v", c.fieldType, c.in, c.want, got)
		}
	}
}

func TestExpandChromePoliciesAdditionalTargetKeys(t *testing.T) {
	in := []interface{}{
		map[string]interface{}{"target_key": "app_id", "target_value": "chrome:abc"},
		map[string]interface{}{"target_key": "profile_id", "target_value": "def"},
	}
	got := expandChromePoliciesAdditionalTargetKeys(in)
	if got["app_id"] != "chrome:abc" || got["profile_id"] != "def" || len(got) != 2 {
		t.Errorf("unexpected map result: %#v", got)
	}
}

func TestExpandChromePoliciesValues(t *testing.T) {
	input := []interface{}{map[string]interface{}{
		"schema_name": "chrome.users.MaxConnectionsPerProxy",
		"schema_values": map[string]interface{}{
			"maxConnectionsPerProxy": jsonMustMarshalToString(8),
		},
	}}
	vals, diags := expandChromePoliciesValues(input)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %#v", diags)
	}
	if len(vals) != 1 {
		t.Fatalf("expected 1 policy value, got %d", len(vals))
	}
	if vals[0].PolicySchema != "chrome.users.MaxConnectionsPerProxy" {
		t.Errorf("unexpected schema name: %s", vals[0].PolicySchema)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(vals[0].Value, &decoded); err != nil {
		t.Fatalf("error unmarshalling stored value: %v", err)
	}
	if decoded["maxConnectionsPerProxy"].(float64) != 8 { // JSON numbers become float64
		t.Errorf("expected stored numeric value 8, got %#v", decoded["maxConnectionsPerProxy"])
	}
}

func jsonMustMarshalToString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
