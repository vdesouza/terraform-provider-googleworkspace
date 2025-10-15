package googleworkspace

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"google.golang.org/api/cloudidentity/v1"
)

func resourceGroupDynamic() *schema.Resource {
	return &schema.Resource{
		Description: "Dynamic Group resource manages Google Workspace Dynamic Groups using the Cloud Identity API. " +
			"Dynamic groups automatically add or remove members based on user attributes. " +
			"This resource requires the `https://www.googleapis.com/auth/cloud-identity.groups` client scope.",

		CreateContext: resourceGroupDynamicCreate,
		ReadContext:   resourceGroupDynamicRead,
		UpdateContext: resourceGroupDynamicUpdate,
		DeleteContext: resourceGroupDynamicDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The resource name of the Group. Format: groups/{group_id}",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"email": {
				Description: "The group's email address. If your account has multiple domains, " +
					"select the appropriate domain for the email address. The email must be unique.",
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"display_name": {
				Description: "The display name of the Group.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"description": {
				Description:      "An extended description to help users determine the purpose of a Group.",
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 4096)),
			},
			"query": {
				Description: "The dynamic membership query for this group. " +
					"Members are automatically added or removed based on this query. " +
					"See https://cloud.google.com/identity/docs/reference/rest/v1/groups.memberships#DynamicGroupQuery",
				Type:     schema.TypeString,
				Required: true,
			},
			"security_group": {
				Description: "If true, adds the cloudidentity.googleapis.com/groups.security label to the group. " +
					"This is an immutable change - once added, the security label cannot be removed.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"locked": {
				Description: "If true, locks the group by adding the cloudidentity.googleapis.com/groups.locked label. " +
					"Locked groups prevent members from being added or removed. This can be toggled on/off.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"labels": {
				Description: "Additional custom label entries that apply to the Group. " +
					"The system labels (dynamic, discussion_forum, security, locked) are managed automatically or via their respective fields. " +
					"The 'dynamic' label is automatically added by the API, and 'discussion_forum' is added by default. " +
					"All label values must be empty strings.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"group_key_id": {
				Description: "The ID of the entity group key.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"group_key_namespace": {
				Description: "The namespace of the entity group key.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"create_time": {
				Description: "The time when the Group was created.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"update_time": {
				Description: "The time when the Group was last updated.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceGroupDynamicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)
	email := d.Get("email").(string)

	log.Printf("[DEBUG] Creating Dynamic Group %q", email)

	cloudIdentityService, diags := client.NewCloudIdentityService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetCloudIdentityGroupsService(cloudIdentityService)
	if diags.HasError() {
		return diags
	}

	// Validate the dynamic query before creating
	query := d.Get("query").(string)
	validationDiags := validateDynamicGroupQuery(query)
	if validationDiags.HasError() {
		return validationDiags
	}

	// Build the group object with labels
	// Note: Do NOT set the dynamic label directly - it's automatically added by the API
	// However, discussion_forum label is required for all groups
	labels := map[string]string{
		"cloudidentity.googleapis.com/groups.discussion_forum": "",
	}

	// Add security label if requested (immutable once added)
	if d.Get("security_group").(bool) {
		labels["cloudidentity.googleapis.com/groups.security"] = ""
	}

	// Add locked label if requested
	if d.Get("locked").(bool) {
		labels["cloudidentity.googleapis.com/groups.locked"] = ""
	}

	// Add any custom labels
	if labelsRaw, ok := d.GetOk("labels"); ok {
		for k, v := range labelsRaw.(map[string]interface{}) {
			labels[k] = v.(string)
		}
	}

	group := &cloudidentity.Group{
		GroupKey: &cloudidentity.EntityKey{
			Id: email,
		},
		Parent: fmt.Sprintf("customers/%s", client.Customer),
		Labels: labels,
		DynamicGroupMetadata: &cloudidentity.DynamicGroupMetadata{
			Queries: []*cloudidentity.DynamicGroupQuery{
				{
					Query: query,
				},
			},
		},
	}

	// Set display name if provided
	if displayName, ok := d.GetOk("display_name"); ok {
		group.DisplayName = displayName.(string)
	} else {
		// Default to email if not provided
		group.DisplayName = email
	}

	// Set description if provided
	if description, ok := d.GetOk("description"); ok {
		group.Description = description.(string)
	}

	// Create the group
	// Note: Dynamic groups must use EMPTY config, not WITH_INITIAL_OWNER
	// because membership is controlled entirely by the query
	createdGroup, err := groupsService.Create(group).InitialGroupConfig("EMPTY").Do()
	if err != nil {
		return diag.Errorf("failed to create dynamic group: %v", err)
	}

	// Extract the group ID from the name (format: groups/{group_id})
	d.SetId(createdGroup.Name)

	log.Printf("[DEBUG] Finished creating Dynamic Group %q with name: %s", email, createdGroup.Name)

	// Wait for the group to be fully created
	err = retryTimeDuration(ctx, d.Timeout(schema.TimeoutCreate), func() error {
		_, retryErr := groupsService.Get(createdGroup.Name).Do()
		if retryErr != nil {
			if isNotFound(retryErr) {
				return fmt.Errorf("group not ready yet")
			}
			return fmt.Errorf("error checking group status: %v", retryErr)
		}
		return nil
	})

	if err != nil {
		return diag.FromErr(fmt.Errorf("timed out waiting for group to be created: %v", err))
	}

	return resourceGroupDynamicRead(ctx, d, meta)
}

func resourceGroupDynamicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	log.Printf("[DEBUG] Reading Dynamic Group with name: %s", d.Id())

	cloudIdentityService, diags := client.NewCloudIdentityService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetCloudIdentityGroupsService(cloudIdentityService)
	if diags.HasError() {
		return diags
	}

	group, err := groupsService.Get(d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, d.Get("email").(string))
	}

	d.Set("name", group.Name)
	d.Set("display_name", group.DisplayName)
	d.Set("description", group.Description)
	d.Set("create_time", group.CreateTime)
	d.Set("update_time", group.UpdateTime)

	if group.GroupKey != nil {
		d.Set("email", group.GroupKey.Id)
		d.Set("group_key_id", group.GroupKey.Id)
		d.Set("group_key_namespace", group.GroupKey.Namespace)
	}

	// Extract the dynamic query
	if group.DynamicGroupMetadata != nil && len(group.DynamicGroupMetadata.Queries) > 0 {
		d.Set("query", group.DynamicGroupMetadata.Queries[0].Query)
	}

	// Check for system labels and set boolean flags
	if group.Labels != nil {
		// Check for security label
		if _, ok := group.Labels["cloudidentity.googleapis.com/groups.security"]; ok {
			d.Set("security_group", true)
		} else {
			d.Set("security_group", false)
		}

		// Check for locked label
		if _, ok := group.Labels["cloudidentity.googleapis.com/groups.locked"]; ok {
			d.Set("locked", true)
		} else {
			d.Set("locked", false)
		}

		// Set custom labels (filter out the system labels)
		customLabels := make(map[string]string)
		systemLabels := map[string]bool{
			"cloudidentity.googleapis.com/groups.dynamic":          true,
			"cloudidentity.googleapis.com/groups.discussion_forum": true,
			"cloudidentity.googleapis.com/groups.security":         true,
			"cloudidentity.googleapis.com/groups.locked":           true,
		}

		for k, v := range group.Labels {
			if !systemLabels[k] {
				customLabels[k] = v
			}
		}

		if len(customLabels) > 0 {
			d.Set("labels", customLabels)
		}
	}

	return diags
}

func resourceGroupDynamicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	log.Printf("[DEBUG] Updating Dynamic Group with name: %s", d.Id())

	cloudIdentityService, diags := client.NewCloudIdentityService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetCloudIdentityGroupsService(cloudIdentityService)
	if diags.HasError() {
		return diags
	}

	// Get the current group
	group, err := groupsService.Get(d.Id()).Do()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get group for update: %v", err))
	}

	var updateMask []string

	if d.HasChange("display_name") {
		group.DisplayName = d.Get("display_name").(string)
		updateMask = append(updateMask, "displayName")
	}

	if d.HasChange("description") {
		group.Description = d.Get("description").(string)
		updateMask = append(updateMask, "description")
	}

	if d.HasChange("query") {
		query := d.Get("query").(string)

		// Validate the new query
		validationDiags := validateDynamicGroupQuery(query)
		if validationDiags.HasError() {
			return validationDiags
		}

		group.DynamicGroupMetadata = &cloudidentity.DynamicGroupMetadata{
			Queries: []*cloudidentity.DynamicGroupQuery{
				{
					Query: query,
				},
			},
		}
		updateMask = append(updateMask, "dynamicGroupMetadata.queries")
	}

	// Handle label changes (security_group, locked, and custom labels)
	if d.HasChange("security_group") || d.HasChange("locked") || d.HasChange("labels") {
		// Build labels map (do NOT include dynamic label - it's automatic)
		// However, discussion_forum label is required for all groups
		labels := map[string]string{
			"cloudidentity.googleapis.com/groups.discussion_forum": "",
		}

		// Add security label if set (note: this is immutable once added)
		if d.Get("security_group").(bool) {
			labels["cloudidentity.googleapis.com/groups.security"] = ""
		}

		// Add or remove locked label
		if d.Get("locked").(bool) {
			labels["cloudidentity.googleapis.com/groups.locked"] = ""
		}

		// Add custom labels
		if labelsRaw, ok := d.GetOk("labels"); ok {
			for k, v := range labelsRaw.(map[string]interface{}) {
				labels[k] = v.(string)
			}
		}

		group.Labels = labels
		updateMask = append(updateMask, "labels")
	}

	if len(updateMask) > 0 {
		_, err = groupsService.Patch(d.Id(), group).UpdateMask(strings.Join(updateMask, ",")).Do()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update dynamic group: %v", err))
		}

		log.Printf("[DEBUG] Finished updating Dynamic Group with name: %s", d.Id())

		// Wait for the update to be reflected
		err = retryTimeDuration(ctx, d.Timeout(schema.TimeoutUpdate), func() error {
			updatedGroup, retryErr := groupsService.Get(d.Id()).Do()
			if retryErr != nil {
				return fmt.Errorf("error checking group status: %v", retryErr)
			}
			// Verify at least one field was updated
			if d.HasChange("display_name") && updatedGroup.DisplayName != d.Get("display_name").(string) {
				return fmt.Errorf("update not reflected yet")
			}
			return nil
		})

		if err != nil {
			log.Printf("[WARN] Update may not be fully reflected: %v", err)
		}
	}

	return resourceGroupDynamicRead(ctx, d, meta)
}

func resourceGroupDynamicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	log.Printf("[DEBUG] Deleting Dynamic Group with name: %s", d.Id())

	cloudIdentityService, diags := client.NewCloudIdentityService()
	if diags.HasError() {
		return diags
	}

	groupsService, diags := GetCloudIdentityGroupsService(cloudIdentityService)
	if diags.HasError() {
		return diags
	}

	_, err := groupsService.Delete(d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, d.Get("email").(string))
	}

	log.Printf("[DEBUG] Finished deleting Dynamic Group with name: %s", d.Id())

	return diags
}

// validateDynamicGroupQuery validates that a dynamic group query has the required structure
func validateDynamicGroupQuery(query string) diag.Diagnostics {
	var diags diag.Diagnostics

	if query == "" {
		return diag.Errorf("query cannot be empty")
	}

	// Basic validation - query should contain field comparisons
	// Common patterns: user.field == 'value', user.field.contains('value')
	if !strings.Contains(query, "user.") && !strings.Contains(query, "resource.") {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Query may be invalid",
			Detail: "Dynamic group queries typically reference 'user.' or 'resource.' fields. " +
				"Example: user.organizations.exists(org, org.department == 'Engineering')",
		})
	}

	// Check for some common operators
	hasOperator := false
	operators := []string{"==", "!=", "contains", "exists", "in", "&&", "||"}
	for _, op := range operators {
		if strings.Contains(query, op) {
			hasOperator = true
			break
		}
	}

	if !hasOperator {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Query may be missing operators",
			Detail: "Dynamic group queries typically use operators like ==, !=, contains, exists, in, &&, ||. " +
				"Example: user.email.contains('@example.com')",
		})
	}

	return diags
}
