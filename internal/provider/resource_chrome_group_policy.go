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

	// Check if we have additional_target_keys
	additionalTargetKeysRaw, hasAdditionalKeys := d.GetOk("additional_target_keys")

	if !hasAdditionalKeys {
		// No additional_target_keys: make individual call for each policy
		log.Printf("[DEBUG] No additional_target_keys - processing policies individually")
		for _, p := range policies {
			policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
				TargetResource: "groups/" + targetID,
			}

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
	} else {
		// Have additional_target_keys: group by target_key
		additionalTargetKeysList := additionalTargetKeysRaw.([]interface{})

		// Group additional_target_keys by their target_key
		keyGroups := make(map[string][]map[string]string)
		for _, k := range additionalTargetKeysList {
			targetKeyDef := k.(map[string]interface{})
			targetKeyName := targetKeyDef["target_key"].(string)
			targetKeyValue := targetKeyDef["target_value"].(string)

			keyGroups[targetKeyName] = append(keyGroups[targetKeyName], map[string]string{
				"key":   targetKeyName,
				"value": targetKeyValue,
			})
		}

		log.Printf("[DEBUG] Grouped additional_target_keys by target_key: %d groups", len(keyGroups))

		// Process each group of target_keys
		for targetKeyName, keyValuePairs := range keyGroups {
			log.Printf("[DEBUG] Processing target_key group: %s with %d values", targetKeyName, len(keyValuePairs))

			// For each value in this target_key group, create requests for all policies
			for _, keyValuePair := range keyValuePairs {
				policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
					TargetResource: "groups/" + targetID,
					AdditionalTargetKeys: map[string]string{
						keyValuePair["key"]: keyValuePair["value"],
					},
				}

				var requests []*chromepolicy.GoogleChromePolicyVersionsV1ModifyGroupPolicyRequest
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
					requests = append(requests, req)
				}

				// Batch all policies for this specific additional_target_key value
				batchReq := &chromepolicy.GoogleChromePolicyVersionsV1BatchModifyGroupPoliciesRequest{
					Requests: requests,
				}

				log.Printf("[DEBUG] Batching %d policies for %s=%s", len(requests), keyValuePair["key"], keyValuePair["value"])

				err := retryTimeDuration(ctx, time.Minute, func() error {
					_, retryErr := chromePoliciesService.Groups.BatchModify(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
					return retryErr
				})

				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	log.Printf("[DEBUG] Finished creating Chrome Policy for groups:%s", targetID)
	d.SetId(targetID)

	return resourceChromeGroupPolicyRead(ctx, d, meta)
}

func resourceChromeGroupPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// client := meta.(*apiClient)

	// _, diags := client.NewChromePolicyService()
	// if diags.HasError() {
	// 	return diags
	// }

	// chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	// if diags.HasError() {
	// 	return diags
	// }

	// log.Printf("[DEBUG] Updating Chrome Policy for groups:%s", d.Id())

	// policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
	// 	TargetResource: "groups/" + d.Id(),
	// }

	// if _, ok := d.GetOk("additional_target_keys"); ok {
	// 	policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	// }

	// old, _ := d.GetChange("policies")

	// For groups, delete old policies before applying new ones
	// Workaround: send only one policy per batch delete call
	// for _, p := range old.([]interface{}) {
	// 	policy := p.(map[string]interface{})
	// 	schemaName := policy["schema_name"].(string)

	// 	deleteReq := &chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{
	// 		PolicyTargetKey: policyTargetKey,
	// 		PolicySchema:    schemaName,
	// 	}

	// 	batchReq := &chromepolicy.GoogleChromePolicyVersionsV1BatchDeleteGroupPoliciesRequest{
	// 		Requests: []*chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{deleteReq},
	// 	}

	// 	err := retryTimeDuration(ctx, time.Minute, func() error {
	// 		_, retryErr := chromePoliciesService.Groups.BatchDelete(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
	// 		return retryErr
	// 	})
	// 	if err != nil {
	// 		// Ignore errors about apps not being installed - this happens when trying to delete
	// 		// policies for apps that were never actually installed/registered
	// 		if isApiErrorWithCode(err, 400) && strings.Contains(err.Error(), "apps are not installed") {
	// 			log.Printf("[DEBUG] Skipping delete for policy %s - app not installed: %v", schemaName, err)
	// 			continue
	// 		}
	// 		return diag.FromErr(err)
	// 	}
	// }

	// Re-run create logic to apply the new set
	diags := resourceChromeGroupPolicyCreate(ctx, d, meta)
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

	// Check if we have additional_target_keys
	additionalTargetKeysRaw, hasAdditionalKeys := d.GetOk("additional_target_keys")

	if !hasAdditionalKeys {
		// No additional target keys - delete policies individually
		policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
			TargetResource: "groups/" + d.Id(),
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
				// Ignore errors about apps not being installed - this happens when trying to delete
				// policies for apps that were never actually installed/registered
				if isApiErrorWithCode(err, 400) && strings.Contains(err.Error(), "apps are not installed") {
					log.Printf("[DEBUG] Skipping delete for policy %s - app not installed: %v", schemaName, err)
					continue
				}
				return diag.FromErr(err)
			}
		}
	} else {
		// Have additional_target_keys: group by target_key
		additionalTargetKeysList := additionalTargetKeysRaw.([]interface{})

		// Group additional_target_keys by their target_key
		keyGroups := make(map[string][]map[string]string)
		for _, k := range additionalTargetKeysList {
			targetKeyDef := k.(map[string]interface{})
			targetKeyName := targetKeyDef["target_key"].(string)
			targetKeyValue := targetKeyDef["target_value"].(string)

			keyGroups[targetKeyName] = append(keyGroups[targetKeyName], map[string]string{
				"key":   targetKeyName,
				"value": targetKeyValue,
			})
		}

		log.Printf("[DEBUG] Grouped additional_target_keys into %d groups", len(keyGroups))

		// For each unique target_key, batch all policies for each unique target_value
		for targetKey, keyValuePairs := range keyGroups {
			log.Printf("[DEBUG] Processing target_key: %s with %d target_values", targetKey, len(keyValuePairs))

			for _, keyValuePair := range keyValuePairs {
				log.Printf("[DEBUG] Batching policies for deletion: target_key=%s, target_value=%s", keyValuePair["key"], keyValuePair["value"])

				policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
					TargetResource: "groups/" + d.Id(),
					AdditionalTargetKeys: map[string]string{
						keyValuePair["key"]: keyValuePair["value"],
					},
				}

				// Batch delete all policies for this specific additional_target_key value
				var deleteRequests []*chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest
				for _, p := range d.Get("policies").([]interface{}) {
					policy := p.(map[string]interface{})
					schemaName := policy["schema_name"].(string)

					deleteRequests = append(deleteRequests, &chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{
						PolicyTargetKey: policyTargetKey,
						PolicySchema:    schemaName,
					})
				}

				batchReq := &chromepolicy.GoogleChromePolicyVersionsV1BatchDeleteGroupPoliciesRequest{
					Requests: deleteRequests,
				}

				log.Printf("[DEBUG] Making BatchDelete call for target_key=%s, target_value=%s with %d policies", keyValuePair["key"], keyValuePair["value"], len(deleteRequests))

				err := retryTimeDuration(ctx, time.Minute, func() error {
					_, retryErr := chromePoliciesService.Groups.BatchDelete(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
					return retryErr
				})
				if err != nil {
					// Ignore 400 errors about apps not being installed - this happens when deleting policies for apps
					// that were uninstalled from the domain
					if isApiErrorWithCode(err, 400) && strings.Contains(err.Error(), "apps are not installed") {
						log.Printf("[DEBUG] Ignoring error about apps not being installed during policy deletion for %s=%s: %v", keyValuePair["key"], keyValuePair["value"], err)
					} else {
						return diag.FromErr(err)
					}
				}
			}
		}
	}

	log.Printf("[DEBUG] Finished deleting Chrome Policy for groups:%s", d.Id())
	return nil
}
