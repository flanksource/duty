## Database models

- The database schema is defined in hcl files in the @schema directory
- atlas-go is used to generate the database models from the hcl files
- Other migration files exists mostly in @views/ directory
- The structs for the models are defined in @models/
- Whenever a new table/view is added or removed, they must be addressed in `dbResourceObjMap` in @rbac/objects.go

## Test Notes

- Always use ginkgo to run the tests.
- Always use gomega for assertions in tests.
- To run a specific test. use `ginkgo -focus "TestName" -r`
