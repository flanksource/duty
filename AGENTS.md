## Database models

- The database schema is defined in hcl files in the @schema directory
- atlas-go is used to generate the database models from the hcl files
- Other migration files exists mostly in @views/ directory
- The structs for the models are defined in @models/
- Whenever a new table/view is added or removed, they must be addressed in `dbResourceObjMap` in @rbac/objects.go

## Database RLS

We use PostgreSQL Row-Level Security (RLS) to enforce multi-tenant access control.
RLS policies filter database rows based on JWT claims passed via PostgREST, ensuring users only see data they have permission to access.

### Policy Patterns

**Direct Policies**: Tables with direct RLS use the `match_scope()` function to evaluate JWT claims against row attributes (tags, agents, names, id).

- Examples: `config_items`, `canaries`, `components`, `playbooks`
- Policy checks row attributes directly using `match_scope(jwt_claims, row.tags, row.agent_id, row.name, row.id)`

**Inherited Policies**: Child tables inherit access control from their parent using `EXISTS` clauses.

- Examples: `checks` (inherits from `canaries`), `playbook_runs` (inherits from `playbooks`)
- Policy: `EXISTS (SELECT 1 FROM parent_table WHERE parent_table.id = child_table.parent_id)`

### Adding RLS to a Table

1. Add RLS enable logic to `@views/9998_rls_enable.sql`
   - Enable RLS on the table
   - Create the policy (either direct with `match_scope()` or inherited with `EXISTS`)
2. Add counterpart disable logic to `@views/9999_rls_disable.sql`
   - Disable RLS on the table
   - Drop the policy
3. Add comprehensive test cases to `@tests/rls_test.go`
   - Test access granted scenarios (various JWT claim combinations)
   - Test access denied scenarios (empty scopes, non-existent resources, conflicting criteria)
   - Test edge cases (wildcards, case sensitivity, empty strings)

### PostgREST JWT Claims Injection

The RLS policies work by injecting JWT claims into PostgreSQL session variables via `request.jwt.claims`. The flow is:

- Go code builds an RLS Payload (scopes for config, component, playbook, canary, view) in `@rls/payload.go`
- `SetPostgresSessionRLS()` serializes the Payload to JSON and executes: `SET request.jwt.claims TO <json>`
- PostgreSQL RLS policies read `(current_setting('request.jwt.claims')::jsonb)` to enforce access control

## Test Notes

- To run the entire test suite, run `make test`.
- To run a specific test. use `ginkgo -focus "TestName" -r`
- To run tests in a package, use ginkgo with `--label-filter='!ignore_local'` flag.
- Always use ginkgo to run tests. Never run `go test` directly.
- Always use `github.com/onsi/gomega` package for assertions.
- Our test suite sets up an embedded Postgres database with data close to production from @tests/fixtures/dummy/all.go.
  Always try to use resources from the dummy dataset before creating one.
- When using gomega with native go tests use this approach

```go
g := gomega.NewWithT(t)
g.Expect(true).To(gomega.Equal(1 == 1))
```

### Comments guidelines

- Only add comments if really really necessary. Do not add comments that simply explain the code.
  - Exception: comments about functions are considered good practice in Go even if they are self-explanatory.

### To Connect to local database

Run

```sh
psql $DB_URL -c "SELECT VERSION()"
```
