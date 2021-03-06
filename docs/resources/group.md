---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "nebraska_group Resource - terraform-provider-nebraska"
subcategory: ""
description: |-
  A group provides a particular release channel to machines and controls various options that manage the update procedure.
---

# nebraska_group (Resource)

A group provides a particular release channel to machines and controls various options that manage the update procedure.

## Example Usage

```terraform
resource "nebraska_application" "demo_app" {
  product_id  = "io.kinvolk.demo"
  name        = "Demo app"
  description = "demo app is used for demo purpose"
}

resource "nebraska_group" "demo_group" {
  name           = "demo group"
  application_id = nebraska_application.demo_app.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `application_id` (String) ID of the application this group belongs to.
- `name` (String) Name of the group.

### Optional

- `channel_id` (String) The channel this group provides.
- `description` (String) A description of the group.
- `id` (String) The ID of this resource.
- `policy_max_updates_per_period` (Number) The maximum number of updates that can be performed within the `policy_period_interval`. Defaults to `1`.
- `policy_office_hours` (Boolean) Only update between 9am and 5pm. Defaults to `false`.
- `policy_period_interval` (String) Period used in combination with `policy_max_updates_per_period`. Defaults to `1 hours`.
- `policy_safe_mode` (Boolean) Safe mode will only update 1 instance at a time, and stop if an update fails. Defaults to `false`.
- `policy_timezone` (String) Timezone used to inform `policy_office_hours`. Defaults to `Asia/Calcutta`.
- `policy_update_timeout` (String) Timeout for updates Defaults to `1 days`.
- `policy_updates_enabled` (Boolean) Enable updates. Defaults to `false`.
- `track` (String) Identifier for clients, filled with the group ID if omitted.

### Read-Only

- `created_ts` (String) Creation timestamp
- `rollout_in_progress` (Boolean) Indicates whether a rollout is currently in progress for this group.


