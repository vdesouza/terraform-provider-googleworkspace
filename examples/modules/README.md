# Module Examples

Runnable examples for the companion modules under [`/modules`](../../modules/). Each subdirectory contains a `main.tf` and a `config/` tree with sample YAML.

These examples use a **local source path** (`source = "../../../modules/<name>"`) so they can be `terraform init`-ed and validated without fetching from GitHub. Real consumers should use the Git source documented in each module's README:

```hcl
source = "git::https://github.com/vdesouza/terraform-provider-googleworkspace.git//modules/<name>?ref=v1.4.0"
```

## Examples

| Directory | Demonstrates |
|---|---|
| [`variables/`](variables/) | Loading YAML variables and exporting `variables_map`. |
| [`groups/`](groups/) | One static group plus one dynamic group. |
| [`assets/`](assets/) | Uploading a placeholder wallpaper to Chrome Policy storage. |
| [`extensions/`](extensions/) | Force-installing a Chrome extension to a group. |
| [`policies/`](policies/) | One group policy plus one OU policy. |
| [`group_priority/`](group_priority/) | Defining a default ordering plus a per-policy ordering. |
