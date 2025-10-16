package googleworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"google.golang.org/api/chromepolicy/v1"
	"google.golang.org/api/googleapi"
)

func resourceChromePolicyFile() *schema.Resource {
	return &schema.Resource{
		Description: "Chrome Policy File resource uploads a file to Chrome Policy API for use in policies like WallpaperImage. " +
			"The uploaded file can be referenced in Chrome policies via the returned download_uri. " +
			"This resource requires the `https://www.googleapis.com/auth/chrome.management.policy` client scope.",

		CreateContext: resourceChromePolicyFileCreate,
		ReadContext:   resourceChromePolicyFileRead,
		UpdateContext: resourceChromePolicyFileUpdate,
		DeleteContext: resourceChromePolicyFileDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"file_path": {
				Description: "The local file path to upload. The file will be uploaded to Chrome Policy API. " +
					"Changes to this path will trigger a new upload.",
				Type:     schema.TypeString,
				Required: true,
			},
			"policy_field": {
				Description: "The fully qualified policy schema and field name this file is uploaded for. " +
					"This is required by the API to validate the content type. " +
					"Example: 'chrome.users.WallpaperImage.value' for wallpaper images.",
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"file_hash": {
				Description: "SHA256 hash of the file content. Automatically computed from file_path. " +
					"Used to detect changes to the file content.",
				Type:     schema.TypeString,
				Computed: true,
			},
			"download_uri": {
				Description: "The download URI of the uploaded file. Use this value in Chrome policies " +
					"that require file references (e.g., WallpaperImage policy).",
				Type:     schema.TypeString,
				Computed: true,
			},
			"upload_time": {
				Description: "The time when the file was uploaded.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceChromePolicyFileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)
	filePath := d.Get("file_path").(string)
	policyField := d.Get("policy_field").(string)

	log.Printf("[DEBUG] Uploading Chrome Policy File from %q for policy field %q", filePath, policyField)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return diag.Errorf("failed to open file %q: %v", filePath, err)
	}
	defer file.Close()

	// Calculate file hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return diag.Errorf("failed to calculate file hash: %v", err)
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return diag.Errorf("failed to reset file pointer: %v", err)
	}

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return diag.Errorf("failed to get file info: %v", err)
	}

	// Create Chrome Policy service
	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	// Upload the file
	mediaService := chromepolicy.NewMediaService(chromePolicyService)

	// Create upload request with the required policy field
	uploadRequest := &chromepolicy.GoogleChromePolicyVersionsV1UploadPolicyFileRequest{
		PolicyField: policyField,
	}

	// Create upload call with customer as parent
	uploadCall := mediaService.Upload(fmt.Sprintf("customers/%s", client.Customer), uploadRequest)

	// Detect content type from file extension
	contentType := "application/octet-stream"
	if filepath.Ext(filePath) == ".jpg" || filepath.Ext(filePath) == ".jpeg" {
		contentType = "image/jpeg"
	} else if filepath.Ext(filePath) == ".png" {
		contentType = "image/png"
	}

	// Set the media upload using ResumableMedia instead of Media,
	// the Google go pkg returns a JSON type error with Media
	uploadCall.ResumableMedia(ctx, file, fileInfo.Size(), contentType)

	log.Printf("[DEBUG] Uploading file %q (size: %d bytes, hash: %s, content-type: %s) for policy field %q",
		filePath, fileInfo.Size(), fileHash, contentType, policyField)

	// Execute the upload
	uploadResponse, err := uploadCall.Do()
	if err != nil {
		// Try to get more details about the error
		if apiErr, ok := err.(*googleapi.Error); ok {
			log.Printf("[ERROR] API Error: Code=%d, Message=%s, Body=%s", apiErr.Code, apiErr.Message, string(apiErr.Body))
			return diag.Errorf("failed to upload file (HTTP %d): %s - %s", apiErr.Code, apiErr.Message, string(apiErr.Body))
		}
		return diag.Errorf("failed to upload file: %v", err)
	}

	if uploadResponse.DownloadUri == "" {
		return diag.Errorf("upload succeeded but no download URI was returned")
	}

	// Use the download URI as the resource ID
	d.SetId(uploadResponse.DownloadUri)
	d.Set("download_uri", uploadResponse.DownloadUri)
	d.Set("file_hash", fileHash)
	d.Set("upload_time", time.Now().Format(time.RFC3339))

	log.Printf("[DEBUG] Successfully uploaded Chrome Policy File with URI: %s", uploadResponse.DownloadUri)

	return diags
}

func resourceChromePolicyFileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// For file uploads, we can't really "read" the file from the API
	// The download URI is the resource identifier and remains valid
	// We can verify the local file still exists and matches the hash

	filePath := d.Get("file_path").(string)
	storedHash := d.Get("file_hash").(string)

	// Check if file still exists
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[WARN] Local file %q no longer exists or is not accessible: %v", filePath, err)
		// Don't fail the read - the uploaded file still exists in Chrome Policy
		return diags
	}
	defer file.Close()

	// Calculate current file hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Printf("[WARN] Failed to calculate file hash: %v", err)
		return diags
	}
	currentHash := hex.EncodeToString(hash.Sum(nil))

	// If hash changed, note it in the state (will trigger update)
	if currentHash != storedHash {
		log.Printf("[DEBUG] File hash changed from %s to %s", storedHash, currentHash)
		d.Set("file_hash", currentHash)
	}

	return diags
}

func resourceChromePolicyFileUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// If file_path or file_hash changed, re-upload the file
	if d.HasChange("file_path") || d.HasChange("file_hash") {
		log.Printf("[DEBUG] File content changed, re-uploading")
		return resourceChromePolicyFileCreate(ctx, d, meta)
	}

	return resourceChromePolicyFileRead(ctx, d, meta)
}

func resourceChromePolicyFileDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// Note: The Chrome Policy API doesn't provide a delete endpoint for uploaded files
	// Files are managed by Google and will be cleaned up automatically
	// We just remove from Terraform state
	log.Printf("[DEBUG] Removing Chrome Policy File from state (API does not support deletion)")

	d.SetId("")

	return diags
}
