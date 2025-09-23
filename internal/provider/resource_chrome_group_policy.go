package googleworkspace

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
	return chromePolicyCreateCommon(ctx, d, meta, targetGroup, "group_id")
}

func resourceChromeGroupPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return chromePolicyUpdateCommon(ctx, d, meta, targetGroup)
}

func resourceChromeGroupPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return chromePolicyReadCommon(ctx, d, meta, targetGroup)
}

func resourceChromeGroupPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return chromePolicyDeleteCommon(ctx, d, meta, targetGroup)
}
