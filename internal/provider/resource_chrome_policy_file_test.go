package googleworkspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceChromePolicyFile_basic(t *testing.T) {
	t.Parallel()

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-image.jpg")

	// Create test file with some content
	testContent := []byte("fake image content for testing")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicyFile_basic(testFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy_file.test", "file_path", testFile),
					resource.TestCheckResourceAttrSet("googleworkspace_chrome_policy_file.test", "download_uri"),
					resource.TestCheckResourceAttrSet("googleworkspace_chrome_policy_file.test", "file_hash"),
					resource.TestCheckResourceAttrSet("googleworkspace_chrome_policy_file.test", "upload_time"),
					testAccCheckChromePolicyFileExists("googleworkspace_chrome_policy_file.test"),
				),
			},
		},
	})
}

func TestAccResourceChromePolicyFile_update(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test-image-1.jpg")
	testFile2 := filepath.Join(tmpDir, "test-image-2.jpg")

	// Create first test file
	testContent1 := []byte("first image content")
	if err := os.WriteFile(testFile1, testContent1, 0644); err != nil {
		t.Fatalf("Failed to create first test file: %v", err)
	}

	// Create second test file with different content
	testContent2 := []byte("second image content - different")
	if err := os.WriteFile(testFile2, testContent2, 0644); err != nil {
		t.Fatalf("Failed to create second test file: %v", err)
	}

	var downloadUri1, downloadUri2, fileHash1, fileHash2 string

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicyFile_basic(testFile1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy_file.test", "file_path", testFile1),
					testAccCheckChromePolicyFileExists("googleworkspace_chrome_policy_file.test"),
					testAccCaptureAttribute("googleworkspace_chrome_policy_file.test", "download_uri", &downloadUri1),
					testAccCaptureAttribute("googleworkspace_chrome_policy_file.test", "file_hash", &fileHash1),
				),
			},
			{
				Config: testAccResourceChromePolicyFile_basic(testFile2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy_file.test", "file_path", testFile2),
					testAccCheckChromePolicyFileExists("googleworkspace_chrome_policy_file.test"),
					testAccCaptureAttribute("googleworkspace_chrome_policy_file.test", "download_uri", &downloadUri2),
					testAccCaptureAttribute("googleworkspace_chrome_policy_file.test", "file_hash", &fileHash2),
					// Verify that the URI and hash changed
					func(s *terraform.State) error {
						if downloadUri1 == downloadUri2 {
							return fmt.Errorf("download_uri should have changed but didn't")
						}
						if fileHash1 == fileHash2 {
							return fmt.Errorf("file_hash should have changed but didn't")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccResourceChromePolicyFile_contentChange(t *testing.T) {
	t.Parallel()

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-image.jpg")

	// Create test file with initial content
	testContent1 := []byte("initial content")
	if err := os.WriteFile(testFile, testContent1, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var fileHash1, fileHash2 string

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicyFile_basic(testFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttribute("googleworkspace_chrome_policy_file.test", "file_hash", &fileHash1),
				),
			},
			{
				// Modify the file content between steps
				PreConfig: func() {
					testContent2 := []byte("modified content - different from initial")
					if err := os.WriteFile(testFile, testContent2, 0644); err != nil {
						t.Fatalf("Failed to modify test file: %v", err)
					}
				},
				Config: testAccResourceChromePolicyFile_basic(testFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCaptureAttribute("googleworkspace_chrome_policy_file.test", "file_hash", &fileHash2),
					// Hash should change when content changes
					func(s *terraform.State) error {
						if fileHash1 == fileHash2 {
							return fmt.Errorf("file_hash should have changed after content modification")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccResourceChromePolicyFile_largeFile(t *testing.T) {
	t.Parallel()

	// Create a larger test file (1MB)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large-test-file.bin")

	// Create 1MB file
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceChromePolicyFile_basic(testFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("googleworkspace_chrome_policy_file.test", "file_path", testFile),
					resource.TestCheckResourceAttrSet("googleworkspace_chrome_policy_file.test", "download_uri"),
					testAccCheckChromePolicyFileExists("googleworkspace_chrome_policy_file.test"),
				),
			},
		},
	})
}

func TestAccResourceChromePolicyFile_missingFile(t *testing.T) {
	t.Parallel()

	// Use a non-existent file path
	nonExistentFile := "/tmp/this-file-does-not-exist.jpg"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceChromePolicyFile_basic(nonExistentFile),
				ExpectError: regexp.MustCompile("failed to open file"),
			},
		},
	})
}

// Helper function to check if the Chrome Policy File resource exists
func testAccCheckChromePolicyFileExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set for resource: %s", resourceName)
		}

		// Check if download_uri is set (which is the ID)
		downloadUri := rs.Primary.Attributes["download_uri"]
		if downloadUri == "" {
			return fmt.Errorf("download_uri is empty for resource: %s", resourceName)
		}

		// Verify it looks like a valid URI
		if len(downloadUri) < 10 {
			return fmt.Errorf("download_uri looks invalid: %s", downloadUri)
		}

		return nil
	}
}

// Helper function to capture attribute values for later comparison
func testAccCaptureAttribute(resourceName, attrName string, target *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		value := rs.Primary.Attributes[attrName]
		*target = value
		return nil
	}
}

// Test configuration templates

func testAccResourceChromePolicyFile_basic(filePath string) string {
	return fmt.Sprintf(`
resource "googleworkspace_chrome_policy_file" "test" {
  file_path    = %q
  policy_field = "chrome.users.WallpaperImage.value"
}
`, filePath)
}

func testAccResourceChromePolicyFile_withPolicy(filePath, orgUnitId string) string {
	return fmt.Sprintf(`
resource "googleworkspace_chrome_policy_file" "test" {
  file_path    = %q
  policy_field = "chrome.users.WallpaperImage.value"
}

resource "googleworkspace_chrome_policy" "wallpaper" {
  org_unit_id = %q

  policies {
    schema_name = "chrome.users.WallpaperImage"
    schema_values = {
      wallpaperImage = jsonencode({
        value = {
          downloadUri = googleworkspace_chrome_policy_file.test.download_uri
        }
      })
    }
  }
}
`, filePath, orgUnitId)
}
