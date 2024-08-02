# duty

Duty (**D**atabase **Ut**ilit**y**) is a home for common database tools, models and helpers used within the mission control suite of projects.

Duty wraps the awesome [atlas](https://github.com/ariga/atlas/) library, and copies some of its code to make use of internal functions.

## Running Tests

`make test` will run tests against a new embedded postgres instance

### Env Vars

- `DUTY_DB_URL`: the `postgres` db, each test will run against a new database called `duty_gingko`
- `TEST_DB_PORT`: will set the port that the test database will run on.
