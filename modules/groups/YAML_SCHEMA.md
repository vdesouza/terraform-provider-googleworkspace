# YAML Configuration Schema Reference

## Quick Reference

### Static Group (with user members)

```yaml
groups:
  group_key:
    email: "group@example.com"
    display_name: "Group Display Name"
    description: "Group description"
    security_group: true
    user_members:
      - "user1@example.com"
      - "user2@example.com"
```

### Static Group (with nested groups)

```yaml
groups:
  group_key:
    email: "group@example.com"
    display_name: "Group Display Name"
    description: "Group description"
    security_group: true
    group_members:
      - "group1@example.com"
      - "group2@example.com"
```

### Static Group (with multiple roles)

```yaml
groups:
  group_key:
    email: "group@example.com"
    display_name: "Group Display Name"
    description: "Group description"
    security_group: true
    user_owners:
      - "owner@example.com"
    user_managers:
      - "manager@example.com"
    user_members:
      - "member1@example.com"
      - "member2@example.com"
    group_members:
      - "team-group@example.com"
```

### Dynamic Group (with custom query)

```yaml
groups:
  group_key:
    email: "group@example.com"
    display_name: "Group Display Name"
    description: "Group description"
    security_group: true
    query: "user.customSchemas.exists(schema, schema.name=='employment' && schema.value.department=='Engineering')"
    suspended_filter: true
```

### Dynamic Group (from org units)

```yaml
groups:
  group_key:
    email: "group@example.com"
    display_name: "Group Display Name"
    description: "Group description"
    security_group: true
    org_units:
      - "Path/To/OrgUnit1"
      - "Path/To/OrgUnit2"
    suspended_filter: true
```

### Dynamic Group (combining query and org units)

```yaml
groups:
  group_key:
    email: "group@example.com"
    display_name: "Group Display Name"
    description: "Group description"
    security_group: true
    query: "user.name.givenName.contains('Engineer')"
    org_units:
      - "Path/To/OrgUnit1"
    suspended_filter: true
```

## Field Descriptions

### Common Fields (All Groups)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `email` | string | ✓ | - | The email address for the group |
| `display_name` | string | | email value | The display name of the group |
| `description` | string | | "" | Description of the group's purpose |
| `security_group` | boolean | | false | Whether this is a security group |
| `aliases` | list | | [] | Email aliases for the group |

### Static Group Member Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `user_members` | list | | [] | List of user emails with MEMBER role |
| `user_managers` | list | | [] | List of user emails with MANAGER role |
| `user_owners` | list | | [] | List of user emails with OWNER role |
| `group_members` | list | | [] | List of group emails with MEMBER role (groups can only be members, not managers/owners) |

### Dynamic Group Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | | - | CEL query for dynamic groups (AND'ed with `org_units` if present) |
| `org_units` | list | | [] | List of org unit paths for dynamic groups (AND'ed with `query` if present) |
| `suspended_filter` | boolean | | true | Add "user.suspended == false" to query |

## Member Roles

The module supports three roles for both users and groups:

- **MEMBER**: Can subscribe to the group, view discussions, and view membership list
- **MANAGER**: Can do everything an OWNER can except make members OWNERS or delete the group (requires Google Groups for Business)
- **OWNER**: Can send messages, add/remove members, change roles, modify settings, and delete the group

## Group Type Logic

The module determines group type based on:

``` text
IF query OR org_units is provided:
  → Create dynamic group
ELSE:
  → Create static group
```

## Query Building for Org Units

When you provide `org_units`, the module builds a query like:

``` CEL
Single OU:
  (user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('123')))
  
Multiple OUs:
  (user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('123'))) || 
  (user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('456')))

With custom query AND org_units (ANDed together):
  (user.name.givenName.contains('Engineer')) && 
  ((user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('123'))) || 
   (user.org_units.exists(org_unit, org_unit.org_unit_id==orgUnitId('456'))))

With suspended_filter = true:
  ((query)) && user.suspended == false
```

This matches users who are **direct or indirect** members of any specified org unit.

## Advanced Examples

### Static Group with Multiple Roles

Different users and groups can have different roles in the same group:

```yaml
groups:
  engineering_team:
    email: "engineering@example.com"
    display_name: "Engineering Team"
    description: "Engineering department with various roles"
    security_group: true
    user_owners:
      - "cto@example.com"
      - "engineering-director@example.com"
    user_managers:
      - "lead-engineer@example.com"
      - "senior-engineer@example.com"
    user_members:
      - "engineer1@example.com"
      - "engineer2@example.com"
      - "intern@example.com"
    group_members:
      - "frontend-team@example.com"
      - "backend-team@example.com"
```

### Nested Groups Hierarchy

Create meta-groups that contain other groups as members:

```yaml
groups:
  # Individual team groups
  frontend_team:
    email: "frontend-team@example.com"
    display_name: "Frontend Team"
    user_members:
      - "frontend1@example.com"
      - "frontend2@example.com"

  backend_team:
    email: "backend-team@example.com"
    display_name: "Backend Team"
    user_members:
      - "backend1@example.com"
      - "backend2@example.com"

  # Meta group containing multiple teams
  all_engineering:
    email: "all-engineering@example.com"
    display_name: "All Engineering"
    description: "Combined group of all engineering teams"
    security_group: true
    user_owners:
      - "cto@example.com"
    group_members:
      - "frontend-team@example.com"
      - "backend-team@example.com"
      - "devops-team@example.com"

  # Leadership group with both individuals and team groups
  leadership:
    email: "leadership@example.com"
    display_name: "Leadership Team"
    security_group: true
    user_owners:
      - "ceo@example.com"
      - "cto@example.com"
    user_managers:
      - "vp-engineering@example.com"
    group_members:
      - "all-engineering@example.com"
      - "all-sales@example.com"
```

### Combining Custom Query with Org Units

When both `query` and `org_units` are provided, they are ANDed together:

```yaml
groups:
  engineering_sanfrancisco:
    email: "engineering-sf@example.com"
    display_name: "Engineering - San Francisco"
    description: "Engineers in San Francisco"
    security_group: true
    query: "user.customSchemas.exists(schema, schema.name=='employment' && schema.value.department=='Eng')"
    org_units:
      - "Engineering/SanFrancisco"
    suspended_filter: true
```

This creates a group with users who match BOTH conditions:

1. Department is Eng (from custom query)
2. Located in Engineering/SanFrancisco org unit

## CEL Query Examples

### Email-based queries

```yaml
# Email contains string
query: "user.email.contains('example.com')"

# Email starts with
query: "user.email.startsWith('contractor-')"

# Email ends with
query: "user.email.endsWith('@example.com')"
```

### Location-based

```yaml
query: "user.customSchemas.exists(schema, schema.name=='employment' && schema.value.location=='San Francisco')"
```

### Custom schema queries

```yaml
# Employee type
query: "user.customSchemas.Employment.employeeType == 'Contractor'"

# Location
query: "user.customSchemas.Location.office == 'San Francisco'"

# Multiple custom fields
query: "user.customSchemas.Employment.department == 'IT' && user.customSchemas.Employment.employeeType == 'FTE'"
```

### Multiple conditions

```yaml
query: "(user.customSchemas.exists(schema, schema.name=='employment' && schema.value.type=='FTE')) && user.suspended == false"
```

## References

- [Dynamic Group Query Language](https://cloud.google.com/identity/docs/how-to/test-query-dynamic-groups)
- [Query Builder/Tester](https://admin.google.com/ac/dgpreview)
