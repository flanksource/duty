---
build: make -B tidy hack/migrate/go.mod && cd hack/migrate && go build -o ../../.github/fixtures/duty-migrate main.go
exec: ./duty-migrate
args: ["--db-url", "postgres://postgres:postgres@localhost:5432/test?sslmode=disable"]
timeout: 10m
---

## Apply migrations

Drives `hack/migrate/main.go` against the matrix Postgres service. The CI workflow runs this fixture twice: once on the merge base (after `actions/checkout` with `ref: main`) and once on the PR head.

| Name           | Exit Code |
|----------------|-----------|
| apply          | 0         |
