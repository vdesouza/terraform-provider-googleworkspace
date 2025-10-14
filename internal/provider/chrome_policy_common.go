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
	"google.golang.org/api/chromepolicy/v1"
)

// chromePolicyTargetKind represents the resource collection prefix used by the API
// (currently "orgunits" or "groups").
type chromePolicyTargetKind string

const (
	targetOrgUnit chromePolicyTargetKind = "orgunits"
	targetGroup   chromePolicyTargetKind = "groups"
)

// chromePolicyCreateCommon contains the shared create logic for both org unit and group chrome policy resources.
func chromePolicyCreateCommon(ctx context.Context, d *schema.ResourceData, meta interface{}, kind chromePolicyTargetKind, idAttr string) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	targetID := strings.TrimPrefix(d.Get(idAttr).(string), "id:")

	log.Printf("[DEBUG] Creating Chrome Policy for %s:%s", kind, targetID)

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: string(kind) + "/" + targetID,
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

	var modifyErr error
	// process group based policies
	if kind == targetGroup {
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

			modifyErr = retryTimeDuration(ctx, time.Minute, func() error {
				_, retryErr := chromePoliciesService.Groups.BatchModify(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
				return retryErr
			})

			if modifyErr != nil {
				return diag.FromErr(modifyErr)
			}
		}
	} else {
		// process org unit based policies
		var requests []*chromepolicy.GoogleChromePolicyVersionsV1ModifyOrgUnitPolicyRequest
		for _, p := range policies {
			var keys []string
			var schemaValues map[string]interface{}
			if err := json.Unmarshal(p.Value, &schemaValues); err != nil {
				return diag.FromErr(err)
			}
			for key := range schemaValues {
				keys = append(keys, key)
			}
			requests = append(requests, &chromepolicy.GoogleChromePolicyVersionsV1ModifyOrgUnitPolicyRequest{
				PolicyTargetKey: policyTargetKey,
				PolicyValue:     p,
				UpdateMask:      strings.Join(keys, ","),
			})
		}

		modifyErr = retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Orgunits.BatchModify(fmt.Sprintf("customers/%s", client.Customer), &chromepolicy.GoogleChromePolicyVersionsV1BatchModifyOrgUnitPoliciesRequest{Requests: requests}).Do()
			return retryErr
		})
	}
	if modifyErr != nil {
		return diag.FromErr(modifyErr)
	}

	log.Printf("[DEBUG] Finished creating Chrome Policy for %s:%s", kind, targetID)
	d.SetId(targetID)

	return chromePolicyReadCommon(ctx, d, meta, kind)
}

// chromePolicyUpdateCommon performs the inherit-then-create update pattern shared by both resources.
func chromePolicyUpdateCommon(ctx context.Context, d *schema.ResourceData, meta interface{}, kind chromePolicyTargetKind) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Updating Chrome Policy for %s:%s", kind, d.Id())

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: string(kind) + "/" + d.Id(),
	}

	if _, ok := d.GetOk("additional_target_keys"); ok {
		policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	}

	old, _ := d.GetChange("policies")

	if kind == targetOrgUnit {
		// For org units, we use inherit-then-create pattern
		var requests []*chromepolicy.GoogleChromePolicyVersionsV1InheritOrgUnitPolicyRequest
		for _, p := range old.([]interface{}) {
			policy := p.(map[string]interface{})
			schemaName := policy["schema_name"].(string)

			requests = append(requests, &chromepolicy.GoogleChromePolicyVersionsV1InheritOrgUnitPolicyRequest{
				PolicyTargetKey: policyTargetKey,
				PolicySchema:    schemaName,
			})
		}

		err := retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Orgunits.BatchInherit(fmt.Sprintf("customers/%s", client.Customer), &chromepolicy.GoogleChromePolicyVersionsV1BatchInheritOrgUnitPoliciesRequest{Requests: requests}).Do()
			return retryErr
		})
		if err != nil {
			return diag.FromErr(err)
		}
	} else if kind == targetGroup {
		// For groups, delete old policies before applying new ones
		var deleteRequests []*chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest
		for _, p := range old.([]interface{}) {
			policy := p.(map[string]interface{})
			schemaName := policy["schema_name"].(string)

			deleteRequests = append(deleteRequests, &chromepolicy.GoogleChromePolicyVersionsV1DeleteGroupPolicyRequest{
				PolicyTargetKey: policyTargetKey,
				PolicySchema:    schemaName,
			})
		}

		err := retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Groups.BatchDelete(fmt.Sprintf("customers/%s", client.Customer), &chromepolicy.GoogleChromePolicyVersionsV1BatchDeleteGroupPoliciesRequest{Requests: deleteRequests}).Do()
			return retryErr
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Re-run create logic to apply the new set.
	diags = chromePolicyCreateCommon(ctx, d, meta, kind, idAttributeForKind(kind))
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Finished updating Chrome Policy for %s:%s", kind, d.Id())
	return diags
}

// chromePolicyReadCommon reads policies for both target kinds.
func chromePolicyReadCommon(ctx context.Context, d *schema.ResourceData, meta interface{}, kind chromePolicyTargetKind) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Getting Chrome Policy for %s:%s", kind, d.Id())

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: string(kind) + "/" + d.Id(),
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
			return diag.FromErr(err)
		}

		if len(resp.ResolvedPolicies) != 1 {
			return diag.Errorf("unexpected number of resolved policies for schema: %s", schemaName)
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

	log.Printf("[DEBUG] Finished getting Chrome Policy for %s:%s", kind, d.Id())
	return nil
}

// chromePolicyDeleteCommon deletes policies for both target kinds.
func chromePolicyDeleteCommon(ctx context.Context, d *schema.ResourceData, meta interface{}, kind chromePolicyTargetKind) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePoliciesService, diags := GetChromePoliciesService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	log.Printf("[DEBUG] Deleting Chrome Policy for %s:%s", kind, d.Id())

	policyTargetKey := &chromepolicy.GoogleChromePolicyVersionsV1PolicyTargetKey{
		TargetResource: string(kind) + "/" + d.Id(),
	}

	if _, ok := d.GetOk("additional_target_keys"); ok {
		policyTargetKey.AdditionalTargetKeys = expandChromePoliciesAdditionalTargetKeys(d.Get("additional_target_keys").([]interface{}))
	}

	var err error
	if kind == targetGroup {
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
			err = retryTimeDuration(ctx, time.Minute, func() error {
				_, retryErr := chromePoliciesService.Groups.BatchDelete(fmt.Sprintf("customers/%s", client.Customer), batchReq).Do()
				return retryErr
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}
	} else {
		var requests []*chromepolicy.GoogleChromePolicyVersionsV1InheritOrgUnitPolicyRequest
		for _, p := range d.Get("policies").([]interface{}) {
			policy := p.(map[string]interface{})
			schemaName := policy["schema_name"].(string)
			requests = append(requests, &chromepolicy.GoogleChromePolicyVersionsV1InheritOrgUnitPolicyRequest{
				PolicyTargetKey: policyTargetKey,
				PolicySchema:    schemaName,
			})
		}

		err = retryTimeDuration(ctx, time.Minute, func() error {
			_, retryErr := chromePoliciesService.Orgunits.BatchInherit(fmt.Sprintf("customers/%s", client.Customer), &chromepolicy.GoogleChromePolicyVersionsV1BatchInheritOrgUnitPoliciesRequest{Requests: requests}).Do()
			return retryErr
		})
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finished deleting Chrome Policy for %s:%s", kind, d.Id())
	return nil
}

// helper to map kind to attribute name
func idAttributeForKind(kind chromePolicyTargetKind) string {
	switch kind {
	case targetOrgUnit:
		return "org_unit_id"
	case targetGroup:
		return "group_id"
	default:
		return "id" // fallback; should not occur
	}
}
