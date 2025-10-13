## Database models

- The database schema is defined in hcl files in the @schema directory
- atlas-go is used to generate the database models from the hcl files
- Other migration files exists mostly in @views/ directory
- The structs for the models are defined in @models/
- Whenever a new table/view is added or removed, they must be addressed in `dbResourceObjMap` in @rbac/objects.go

## Test Notes

- To run the entire test suite, run `make test`.
- To run a specific test. use `ginkgo -focus "TestName" -r`
- To run tests in a package, use ginkgo with `--label-filter='!ignore_local'` flag.
- Always use ginkgo to run tests. Never run `go test` directly.
- Always use `github.com/onsi/gomega` package for assertions.
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
