package googleworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"google.golang.org/api/chromepolicy/v1"
)

func resourceChromeGroupPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Chrome Policy resource in the Terraform Googleworkspace provider. " +
			"Chrome Policy Schema resides under the `https://www.googleapis.com/auth/chrome.management.policy` client scope.",

		CreateContext: resourceChromeGroupPolicyCreate,
		UpdateContext: resourceChromeGroupPolicyUpdate,
		ReadContext:   resourceChromeGroupPolicyRead,
		DeleteContext: resourceChromeGroupPolicyDelete,

		Schema: map[string]*schema.Schema{
			"group_id": {
				Description: "The target group on which this policy is applied.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"additional_target_keys": {
				Description: "Additional target keys for policies.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_key": {
							Description: "The target key name.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"target_value": {
							Description: "The target key value.",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"policies": {
				Description: "Policies to set for the org unit",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"schema_name": {
							Description: "The full qualified name of the policy schema.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"schema_values": {
							Description: "JSON encoded map that represents key/value pairs that " +
								"correspond to the given schema. ",
							Type:     schema.TypeMap,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateDiagFunc: validation.ToDiagFunc(
									validation.StringIsJSON,
								),
							},
						},
					},
				},
			},
		},
	}
}

func resourceChromeGroupPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	targetID := d.Get("group_id").(string)

	log.Printf("[DEBUG] Creating Chrome Policy for groups:%s", targetID)

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: "groups/" + targetID,
	}

	if _, ok := d.GetOk("additional_target_keys"); ok {
		policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	}

	diags = validateChromePolicies(ctx, d, client)
	if diags.HasError() {
		return diags
	}

	policies, diags := expandChromePoliciesValues(d.Get("policies").([]interface{}))
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Expanded policies: %+v", policies)

	// Process group based policies
	// Make individual API calls for each policy instead of batching
	// This works around an issue where batch requests fail with multiple policies
	for _, p := range policies {
		var keys []string
		var schemaValues map[string]interface{}
		if err := json.Unmarshal(p.Value, &schemaValues); err != nil {
			return diag.FromErr(err)
		}
		for key := range schemaValues {
			keys = append(keys, key)
		}
		req := &chromepolicy.GoogleChromePolicyVersionsV1ModifyGroupPolicyRequest{
			PolicyTargetKey: policyTargetKey,
			PolicyValue:     p,
			UpdateMask:      strings.Join(keys, ","),
		}
		log.Printf("[DEBUG] Group policy request: %+v", req)
		log.Printf("[DEBUG] Group policy value: %+v", p)
		log.Printf("[DEBUG] Group policy value (raw bytes): %s", string(p.Value))
		log.Printf("[DEBUG] Group policy schema: %s", p.PolicySchema)
		log.Printf("[DEBUG] Update mask: %s", strings.Join(keys, ","))

		// Make individual call
		batchReq := &chromepolicy.GoogleChromePolicyVersionsV1BatchModifyGroupPoliciesRequest{
			Requests: []*chromepolicy.GoogleChromePolicyVersionsV1ModifyGroupPolicyRequest{req},
		}

		err := retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Groups.BatchModify(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
			return retryErr
		})

		if err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] Finished creating Chrome Policy for groups:%s", targetID)
	d.SetId(targetID)

	return resourceChromeGroupPolicyRead(ctx, d, meta)
}

func resourceChromeGroupPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Updating Chrome Policy for groups:%s", d.Id())

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: "groups/" + d.Id(),
	}

	if _, ok := d.GetOk("additional_target_keys"); ok {
		policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	}

	old, _ := d.GetChange("policies")

	// For groups, delete old policies before applying new ones
	// Workaround: send only one policy per batch delete call
	for _, p := range old.([]interface{}) {
		policy := p.(map[string]interface{})
		schemaName := policy["schema_name"].(string)

		deleteReq := &chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{
			PolicyTargetKey: policyTargetKey,
			PolicySchema:    schemaName,
		}

		batchReq := &chromepolicy.GoogleChromePolicyVersionsV1BatchDeleteGroupPoliciesRequest{
			Requests: []*chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{deleteReq},
		}

		err := retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Groups.BatchDelete(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
			return retryErr
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Re-run create logic to apply the new set
	diags = resourceChromeGroupPolicyCreate(ctx, d, meta)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Finished updating Chrome Policy for groups:%s", d.Id())
	return diags
}

func resourceChromeGroupPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Getting Chrome Policy for groups:%s", d.Id())

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: "groups/" + d.Id(),
	}

	if _, ok := d.GetOk("additional_target_keys"); ok {
		policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	}

	policiesObj := []*chromepolicy.GoogleChromePolicyVersionsV1PolicyValue{}
	for _, p := range d.Get("policies").([]interface{}) {
		policy := p.(map[string]interface{})
		schemaName := policy["schema_name"].(string)

		var resp *chromepolicy.GoogleChromePolicyVersionsV1ResolveResponse
		err := retryTimeDuration(ctx, time.Minute, func() error {
			var retryErr error
			resp, retryErr = chromePoliciesService.Resolve(fmt.Sprintf("customers/%s", client.Customer), &chromepolicy.GoogleChromePolicyVersionsV1ResolveRequest{
				PolicySchemaFilter: schemaName,
				PolicyTargetKey:    policyTargetKey,
			}).Do()
			return retryErr
		})
		if err != nil {
			// Check if it's a 404 error - the group or policy was deleted outside of Terraform
			return handleNotFoundError(err, d, fmt.Sprintf("Chrome Group Policy %s", d.Id()))
		}

		// Handle cases where policy might not exist or has been deleted
		// This can happen when removing a resource from terraform config
		if len(resp.ResolvedPolicies) == 0 {
			log.Printf("[DEBUG] No resolved policies found for schema %s - policy may have been deleted", schemaName)
			// Skip this policy - it doesn't exist in Google anymore
			continue
		}

		if len(resp.ResolvedPolicies) != 1 {
			log.Printf("[WARN] Expected 1 resolved policy for schema %s, got %d", schemaName, len(resp.ResolvedPolicies))
			// Use the first policy if multiple are returned
		}

		value := resp.ResolvedPolicies[0].Value
		policiesObj = append(policiesObj, value)
	}

	policies, diags := flattenChromePolicies(ctx, policiesObj, client)
	if diags.HasError() {
		return diags
	}

	if err := d.Set("policies", policies); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finished getting Chrome Policy for groups:%s", d.Id())
	return nil
}

func resourceChromeGroupPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Deleting Chrome Policy for groups:%s", d.Id())

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: "groups/" + d.Id(),
	}

	if _, ok := d.GetOk("additional_target_keys"); ok {
		policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	}

	// Workaround: send only one policy per batch delete call
	for _, p := range d.Get("policies").([]interface{}) {
		policy := p.(map[string]interface{})
		schemaName := policy["schema_name"].(string)
		deleteReq := &chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{
			PolicyTargetKey: policyTargetKey,
			PolicySchema:    schemaName,
		}
		batchReq := &chromepolicy.GoogleChromePolicyVersionsV1BatchDeleteGroupPoliciesRequest{
			Requests: []*chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{deleteReq},
		}
		err := retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Groups.BatchDelete(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
			return retryErr
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] Finished deleting Chrome Policy for groups:%s", d.Id())
	return nil
}
