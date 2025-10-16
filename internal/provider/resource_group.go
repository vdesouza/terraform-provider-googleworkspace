package googleworkspace

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/googleapi"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Group resource manages Google Workspace Groups. Group resides under the " +
			"`https://www.googleapis.com/auth/admin.directory.group` client scope.",

		CreateContext: resourceGroupCreate,
		ReadContext:   resourceGroupRead,
		UpdateContext: resourceGroupUpdate,
		DeleteContext: resourceGroupDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The unique ID of a group. A group id can be used as a group request URI's groupKey.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"email": {
				Description: "The group's email address. If your account has multiple domains," +
					"select the appropriate domain for the email address. The email must be unique.",
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Description: "The group's display name.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"description": {
				Description: "An extended description to help users determine the purpose of a group." +
					"For example, you can include information about who should join the group," +
					"the types of messages to send to the group, links to FAQs about the group, or related groups.",
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 4096)),
			},
			"admin_created": {
				Description: "Value is true if this group was created by an administrator rather than a user.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"direct_members_count": {
				Description: "The number of users that are direct members of the group." +
					"If a group is a member (child) of this group (the parent)," +
					"members of the child group are not counted in the directMembersCount property of the parent group.",
				Type:     schema.TypeInt,
				Computed: true,
			},
			"etag": {
				Description: "ETag of the resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"aliases": {
				Description: "asps.list of group's email addresses.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"non_editable_aliases": {
				Description: "asps.list of the group's non-editable alias email addresses that are outside of the " +
					"account's primary domain or subdomains. These are functioning email addresses used by the group.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"security_group": {
				Description: "If true, adds the cloudidentity.googleapis.com/groups.security label to the group via the Cloud Identity API. " +
					"This is an immutable change - once added, the security label cannot be removed. " +
					"Requires the cloud-identity.groups OAuth scope.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*apiClient)

	email := d.Get("email").(string)
	log.Printf("[DEBUG] Creating Group %q: %#v", email, email)

	directoryService, diags := client.NewDirectoryService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetGroupsService(directoryService)
	if diags.HasError() {
		return diags
	}

	groupObj := directory.Group{
		Email:       d.Get("email").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	group, err := groupsService.Insert(&groupObj).Do()
	if err != nil {
		return diag.FromErr(err)
	}

	// The etag changes with each insert, so we want to monitor how many changes we should see
	// when we're checking for eventual consistency
	numInserts := 1

	d.SetId(group.Id)

	aliases := d.Get("aliases.#").(int)

	if aliases > 0 {
		aliasesService, diags := GetGroupAliasService(groupsService)
		if diags.HasError() {
			return diags
		}

		for i := 0; i < aliases; i++ {
			aliasObj := directory.Alias{
				Alias: d.Get(fmt.Sprintf("aliases.%d", i)).(string),
			}

			_, err := aliasesService.Insert(d.Id(), &aliasObj).Do()
			if err != nil {
				return diag.FromErr(err)
			}
			numInserts += 1
		}
	}

	// INSERT will respond with the Group that will be created, however, it is eventually consistent
	// After INSERT, the etag is updated along with the Group (and any aliases),
	// once we get a consistent etag, we can feel confident that our Group is also consistent
	cc := consistencyCheck{
		resourceType: "group",
		timeout:      d.Timeout(schema.TimeoutCreate),
	}
	err = retryTimeDuration(ctx, d.Timeout(schema.TimeoutCreate), func() error {
		var retryErr error

		if cc.reachedConsistency(numInserts) {
			return nil
		}

		newGroup, retryErr := groupsService.Get(d.Id()).IfNoneMatch(cc.lastEtag).Do()
		if googleapi.IsNotModified(retryErr) {
			cc.currConsistent += 1
		} else if isNotFound(retryErr) {
			// group was not found yet therefore setting currConsistent back to null value
			cc.currConsistent = 0
		} else if retryErr != nil {
			return fmt.Errorf("unexpected error during retries of %s: %s", cc.resourceType, retryErr)
		} else {
			cc.handleNewEtag(newGroup.Etag)
		}

		return fmt.Errorf("timed out while waiting for %s to be inserted", cc.resourceType)
	})

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finished creating Group %q: %#v", d.Id(), email)

	// Handle security label if requested
	if d.Get("security_group").(bool) {
		log.Printf("[DEBUG] Adding security label to Group %q", d.Id())
		if err := addSecurityLabelToGroup(ctx, client, group.Email); err != nil {
			// Log the error but continue to read the actual state
			log.Printf("[WARN] Failed to add security label to group %s: %v", group.Email, err)
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Failed to add security label to group",
				Detail:   fmt.Sprintf("Group was created successfully but failed to add security label: %v. The state will reflect the actual label status.", err),
			})
		}
	}

	// Always read to ensure state matches reality, especially for security label
	readDiags := resourceGroupRead(ctx, d, meta)
	diags = append(diags, readDiags...)
	return diags
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*apiClient)

	directoryService, diags := client.NewDirectoryService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetGroupsService(directoryService)
	if diags.HasError() {
		return diags
	}

	group, err := groupsService.Get(d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, d.Get("email").(string))
	}

	d.Set("email", group.Email)
	d.Set("name", group.Name)
	d.Set("description", group.Description)
	d.Set("admin_created", group.AdminCreated)
	d.Set("direct_members_count", group.DirectMembersCount)
	d.Set("aliases", group.Aliases)
	d.Set("non_editable_aliases", group.NonEditableAliases)
	d.Set("etag", group.Etag)

	// Always check if the group has a security label via Cloud Identity API
	// This ensures the state always reflects reality
	hasSecurityLabel, err := checkSecurityLabel(ctx, client, group.Email)
	if err != nil {
		log.Printf("[WARN] Failed to check security label for group %s: %v", group.Email, err)
		// Add a warning diagnostic but don't fail the read
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Unable to verify security label status",
			Detail:   fmt.Sprintf("Could not verify if group has security label via Cloud Identity API: %v. The security_group attribute may not reflect the current state.", err),
		})
		// Keep the current state value if we can't verify
	} else {
		// Always update the state to match reality
		d.Set("security_group", hasSecurityLabel)
		log.Printf("[DEBUG] Group %s security label status: %v", group.Email, hasSecurityLabel)
	}

	d.SetId(group.Id)

	return diags
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*apiClient)

	email := d.Get("email").(string)
	log.Printf("[DEBUG] Updating Group %q: %#v", d.Id(), email)

	directoryService, diags := client.NewDirectoryService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetGroupsService(directoryService)
	if diags.HasError() {
		return diags
	}

	groupObj := directory.Group{}

	if d.HasChange("email") {
		groupObj.Email = d.Get("email").(string)
	}

	if d.HasChange("name") {
		groupObj.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		groupObj.Description = d.Get("description").(string)
	}

	numInserts := 0
	if d.HasChange("aliases") {
		old, new := d.GetChange("aliases")
		oldAliases := listOfInterfacestoStrings(old.([]interface{}))
		newAliases := listOfInterfacestoStrings(new.([]interface{}))

		aliasesService, diags := GetGroupAliasService(groupsService)
		if diags.HasError() {
			return diags
		}

		// Remove old aliases that aren't in the new aliases list
		for _, alias := range oldAliases {
			if stringInSlice(newAliases, alias) {
				continue
			}

			err := aliasesService.Delete(d.Id(), alias).Do()
			if err != nil {
				return diag.FromErr(err)
			}
		}

		// Insert all new aliases that weren't previously in state
		for _, alias := range newAliases {
			if stringInSlice(oldAliases, alias) {
				continue
			}

			aliasObj := directory.Alias{
				Alias: alias,
			}

			_, err := aliasesService.Insert(d.Id(), &aliasObj).Do()
			if err != nil {
				return diag.FromErr(err)
			}
			numInserts += 1
		}
	}

	if &groupObj != new(directory.Group) {
		group, err := groupsService.Update(d.Id(), &groupObj).Do()
		if err != nil {
			return diag.FromErr(err)
		}
		numInserts += 1

		d.SetId(group.Id)
	}

	// UPDATE will respond with the Group that will be created, however, it is eventually consistent
	// After UPDATE, the etag is updated along with the Group (and any aliases),
	// once we get a consistent etag, we can feel confident that our Group is also consistent
	cc := consistencyCheck{
		resourceType: "group",
		timeout:      d.Timeout(schema.TimeoutUpdate),
	}
	err := retryTimeDuration(ctx, d.Timeout(schema.TimeoutUpdate), func() error {
		var retryErr error

		if cc.reachedConsistency(numInserts) {
			return nil
		}

		newGroup, retryErr := groupsService.Get(d.Id()).IfNoneMatch(cc.lastEtag).Do()
		if googleapi.IsNotModified(retryErr) {
			cc.currConsistent += 1
		} else if retryErr != nil {
			return fmt.Errorf("unexpected error during retries of %s: %s", cc.resourceType, retryErr)
		} else {
			cc.handleNewEtag(newGroup.Etag)
		}

		return fmt.Errorf("timed out while waiting for %s to be updated", cc.resourceType)
	})

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finished creating Group %q: %#v", d.Id(), email)

	// Handle security label changes
	if d.HasChange("security_group") {
		old, new := d.GetChange("security_group")
		oldValue := old.(bool)
		newValue := new.(bool)

		// Prevent removing the security label (it's immutable)
		if oldValue && !newValue {
			return append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Cannot remove security label from group",
				Detail:   "The security label is immutable and cannot be removed once added. You can only change it from false to true.",
			})
		}

		// Add the security label if changed from false to true
		if !oldValue && newValue {
			log.Printf("[DEBUG] Adding security label to Group %q", d.Id())
			if err := addSecurityLabelToGroup(ctx, client, email); err != nil {
				// Log the error but continue to read the actual state
				log.Printf("[WARN] Failed to add security label to group %s: %v", email, err)
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Failed to add security label to group",
					Detail:   fmt.Sprintf("Group was updated successfully but failed to add security label: %v. The state will reflect the actual label status.", err),
				})
			}
		}
	}

	// Always read to ensure state matches reality, especially for security label
	readDiags := resourceGroupRead(ctx, d, meta)
	diags = append(diags, readDiags...)
	return diags
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*apiClient)

	email := d.Get("email").(string)
	log.Printf("[DEBUG] Deleting Group %q: %#v", d.Id(), email)

	directoryService, diags := client.NewDirectoryService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetGroupsService(directoryService)
	if diags.HasError() {
		return diags
	}

	err := groupsService.Delete(d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, d.Get("email").(string))
	}

	log.Printf("[DEBUG] Finished deleting Group %q: %#v", d.Id(), email)

	return diags
}

// addSecurityLabelToGroup adds the security label to a group using the Cloud Identity API
func addSecurityLabelToGroup(ctx context.Context, client *apiClient, groupEmail string) error {
	cloudIdentityService, diags := client.NewCloudIdentityService()
	if diags.HasError() {
		return fmt.Errorf("failed to create Cloud Identity service: %v", diags)
	}

	groupsService, diags := GetCloudIdentityGroupsService(cloudIdentityService)
	if diags.HasError() {
		return fmt.Errorf("failed to get Cloud Identity groups service: %v", diags)
	}

	// Use the lookup API to find the group by its email address
	// This is more reliable than search and is the recommended approach
	lookupResp, err := groupsService.Lookup().
		GroupKeyId(groupEmail).
		Do()
	if err != nil {
		return fmt.Errorf("failed to lookup group: %v", err)
	}

	if lookupResp.Name == "" {
		return fmt.Errorf("group not found in Cloud Identity")
	}

	// Get the full group details to check and update labels
	group, err := groupsService.Get(lookupResp.Name).Do()
	if err != nil {
		return fmt.Errorf("failed to get group details: %v", err)
	}

	// Check if the security label already exists
	if group.Labels != nil {
		if _, exists := group.Labels["cloudidentity.googleapis.com/groups.security"]; exists {
			log.Printf("[DEBUG] Security label already exists on group %s", groupEmail)
			return nil
		}
	}

	// Add the security label
	if group.Labels == nil {
		group.Labels = make(map[string]string)
	}
	group.Labels["cloudidentity.googleapis.com/groups.security"] = ""

	// Update the group with the new label
	updateMask := "labels"
	_, err = groupsService.Patch(group.Name, group).UpdateMask(updateMask).Do()
	if err != nil {
		return fmt.Errorf("failed to add security label: %v", err)
	}

	log.Printf("[DEBUG] Successfully added security label to group %s", groupEmail)
	return nil
}

// checkSecurityLabel checks if a group has the security label via the Cloud Identity API
func checkSecurityLabel(ctx context.Context, client *apiClient, groupEmail string) (bool, error) {
	cloudIdentityService, diags := client.NewCloudIdentityService()
	if diags.HasError() {
		return false, fmt.Errorf("failed to create Cloud Identity service: %v", diags)
	}

	groupsService, diags := GetCloudIdentityGroupsService(cloudIdentityService)
	if diags.HasError() {
		return false, fmt.Errorf("failed to get Cloud Identity groups service: %v", diags)
	}

	// Use the lookup API to find the group by its email address
	// This is more reliable than search and is the recommended approach
	lookupResp, err := groupsService.Lookup().
		GroupKeyId(groupEmail).
		Do()
	if err != nil {
		return false, fmt.Errorf("failed to lookup group: %v", err)
	}

	if lookupResp.Name == "" {
		return false, fmt.Errorf("group not found in Cloud Identity")
	}

	// Get the full group details to check labels
	group, err := groupsService.Get(lookupResp.Name).Do()
	if err != nil {
		return false, fmt.Errorf("failed to get group details: %v", err)
	}

	// Check if the security label exists
	if group.Labels != nil {
		if _, exists := group.Labels["cloudidentity.googleapis.com/groups.security"]; exists {
			return true, nil
		}
	}

	return false, nil
}
