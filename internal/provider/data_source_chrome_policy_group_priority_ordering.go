package googleworkspace

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/api/chromepolicy/v1"
)

func dataSourceChromePolicyGroupPriorityOrdering() *schema.Resource {
	return &schema.Resource{
		Description: "Chrome Policy Group Priority Ordering data source in the Terraform Googleworkspace provider. " +
			"Use this data source to retrieve the priority ordering of groups for a specific Chrome policy.",

		ReadContext: dataSourceChromePolicyGroupPriorityOrderingRead,

		Schema: map[string]*schema.Schema{
			"policy_schema": {
				Description: "The full qualified name of the policy schema.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"policy_namespace": {
				Description: "The namespace of the policy type for the request.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"policy_target_key": {
				Description: "The target resource for which to retrieve the group priority ordering. " +
					"The target app must be supplied in additional_target_key_names.",
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_resource": {
							Description: "The target resource on which this policy is applied (e.g., 'orgunits/123').",
							Type:        schema.TypeString,
							Required:    true,
						},
						"additional_target_key_names": {
							Description: "Map of additional target keys. The target app must be supplied here.",
							Type:        schema.TypeMap,
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"group_ids": {
				Description: "Ordered list of group IDs. Groups earlier in the list have higher priority " +
					"and their policies will override those of groups later in the list.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"id": {
				Description: "Identifier for the data source.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceChromePolicyGroupPriorityOrderingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	chromePolicyService, diags := client.NewChromePolicyService()
	if diags.HasError() {
		return diags
	}

	chromePolicyGroupsService, diags := GetChromePolicyGroupsService(chromePolicyService)
	if diags.HasError() {
		return diags
	}

	policySchema := d.Get("policy_schema").(string)
	policyNamespace := d.Get("policy_namespace").(string)
	policyTargetKey := expandPolicyTargetKey(d.Get("policy_target_key").([]interface{}))

	log.Printf("[DEBUG] Reading Chrome Policy Group Priority Ordering data source for schema: %s", policySchema)

	req := &chromepolicy.GoogleChromePolicyVersionsV1ListGroupPriorityOrderingRequest{
		PolicyTargetKey: policyTargetKey,
		PolicyNamespace: policyNamespace,
		PolicySchema:    policySchema,
	}

	var resp *chromepolicy.GoogleChromePolicyVersionsV1ListGroupPriorityOrderingResponse
	err := retryTimeDuration(ctx, time.Minute, func() error {
		var retryErr error
		resp, retryErr = chromePolicyGroupsService.ListGroupPriorityOrdering(fmt.Sprintf("customers/%s", client.Customer), req).Do()
		return retryErr
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("group_ids", resp.GroupIds); err != nil {
		return diag.FromErr(err)
	}

	// Set the ID based on the policy schema and target resource
	id := generateChromePolicyGroupPriorityOrderingID(policySchema, policyTargetKey.TargetResource)
	d.SetId(id)

	log.Printf("[DEBUG] Finished reading Chrome Policy Group Priority Ordering data source: %s", id)

	return nil
}
