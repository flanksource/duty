# duty

Duty (**D**atabase **Ut**ilit**y**) is a home for common database tools, models and helpers used within the mission control suite of projects.

Duty wraps the awesome [atlas](https://github.com/ariga/atlas/) library, and copies some of its code to make use of internal functions.


## Running Tests

1. `make test` will run tests against a new embedded postgres instance

If you set `DUTY_DB_URL` environment variable to the `postgres` db, each test will run against a new database called `duty_gingko`
