# Duty Properties

Duty uses runtime properties for behavior that should be adjustable without
changing code. These properties are plain string key/value pairs. They are
separate from model `properties` fields such as `config_items.properties`,
`components.properties`, or connection `properties`, which are resource
metadata stored in JSON columns.

The machine-readable schema for JSON/YAML maps of these runtime properties is
[PROPERTIES.schema.json](/Users/moshe/go/src/github.com/flanksource/duty/PROPERTIES.schema.json).
The schema is intentionally kept outside `schema/openapi` because that
directory is generated.

## Setting Properties

### CLI

Applications that bind `github.com/flanksource/commons/properties.BindFlags`
accept `-P` / `--properties`:

```sh
duty-command -P log.level=debug -P query.log=true
duty-command --properties log.level.http=trace
```

CLI properties are held in the process-local commons property store.

### Environment

Duty does not provide a generic environment-variable-to-property mapper for
runtime properties. To set a runtime property from the environment, pass it
through the CLI, a properties file, DB, or code that calls `properties.Set`.

```sh
duty-command -P query.log="$QUERY_LOG"
```

Startup configuration has explicit environment variables and environment
indirection:

| Env var | Purpose |
| --- | --- |
| `DB_URL` | Default value for `--db` if `--db` is not set. |
| `PGRST_JWT_SECRET` | Default value for `--postgrest-jwt-secret` if the flag is not set. |
| `PGRST_VERSION` | Overrides the bundled PostgREST version. |
| `PGRST_ARCH` | Overrides the PostgREST binary architecture. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Default OpenTelemetry collector endpoint. |
| `OTEL_LABELS` | Comma-separated OpenTelemetry resource labels, `key=value,key2=value2`. |
| `POD_NAMESPACE` | Kubernetes namespace used by leader election. |
| `MC_HOSTNAME_OVERRIDE` | Hostname override used by leader election. |
| `DUTY_DB_DISABLE_RLS` | Used by tests and `hack/migrate`; disables RLS when set to `true`. |
| `DUTY_DB_URL`, `DUTY_DB_CREATE`, `DUTY_DB_DATA_DIR` | Test database setup only. |
| `DUTY_BENCH_SIZES` | Benchmark size list only. |
| `KUBECONFIG` | Kubernetes config path used by the Kubernetes client. |

String startup flags are also passed through `api.Config.ReadEnv()`: when a
flag value is the name of an environment variable, Duty uses that variable's
value. For example, `--db DUTY_DB_URL` reads `DUTY_DB_URL`.

### File

Code may call `properties.LoadFile("duty.properties")`. The file format is:

```properties
# comments are allowed
log.level=debug
query.log=true
topology.query.timeout=45s
```

The commons property loader watches the loaded file and reloads it on changes.
There is no built-in `--properties-file` flag in this package.

Known embedding applications load these default files:

| Application | Default file |
| --- | --- |
| incident-commander / mission-control | `mission-control.properties` |
| config-db | `config-db.properties` |
| canary-checker | `canary-checker.properties` |

### Database

Context-aware properties are read from the `properties` table:

```sql
INSERT INTO properties (name, value)
VALUES ('query.log', 'true')
ON CONFLICT (name) DO UPDATE SET value = excluded.value;
```

From Go, use:

```go
context.UpdateProperty(ctx, "query.log", "true")
context.UpdateProperties(ctx, map[string]string{"query.log": "true"})
```

Database properties are cached in process for 15 minutes. `UpdateProperty` and
`UpdateProperties` clear the cache after writing.

### Annotations

For object-scoped lookup through `ctx.Properties()`, add annotations with one
of these prefixes:

```yaml
metadata:
  annotations:
    mission-control/query.log: "true"
    canary-checker/topology.query.timeout: 45s
```

The prefix is stripped before lookup, so `mission-control/query.log` sets
`query.log`. Child objects override parent objects.

Logging annotations also accept unprefixed forms in addition to the prefixed
forms:

```yaml
metadata:
  annotations:
    log.level: debug
    trace: "true"
    debug: "true"
```

## Precedence

For context-aware properties resolved by `ctx.Properties()`:

1. Process-local commons properties: CLI `-P`, loaded properties file, or code
   that calls `properties.Set`.
2. Object annotations, with child object annotations overriding parent object
   annotations.
3. Database rows in the `properties` table.
4. The hard-coded default at the call site.

Process-global properties that call `properties.String`, `properties.Int`,
`properties.Duration`, or `properties.On` directly do not read DB rows or
annotations. They only see the process-local commons property store.

Boolean values for `ctx.Properties().On` are true when set to `true`,
`enabled`, or `on`. `ctx.Properties().Off` treats `false`, `disabled`, and
`off` as off. The lower-level commons `properties.On` only treats `true` as
true.

## Startup Flags

These are not runtime properties, but they are the main Duty startup settings:

| Flag | Default | Notes |
| --- | --- | --- |
| `--db` | `DB_URL` | PostgreSQL connection string. The default is resolved from `DB_URL`. |
| `--db-schema` | `public` | PostgreSQL schema. |
| `--postgrest-uri` | `http://localhost:3000` | Localhost starts an embedded PostgREST process. Empty disables PostgREST. |
| `--postgrest-log-level` | `info` | PostgREST log level. |
| `--postgrest-jwt-secret` | `PGRST_JWT_SECRET` | JWT secret. The default is resolved from `PGRST_JWT_SECRET`. |
| `--disable-postgrest` | varies | Deprecated; use `--postgrest-uri ''`. |
| `--postgrest-role` | `postgrest_api` | Authenticated PostgREST database role. |
| `--postgrest-anon-role` | `postgrest_anon` | Unauthenticated PostgREST database role. |
| `--postgrest-max-rows` | `2000` | Hard row limit for PostgREST. |
| `--db-log-level` | `error` | GORM log level: `trace`, `debug`, `info`, `error`. |
| `--disable-kubernetes` | `false` | Disable Kubernetes integration. |
| `--db-metrics` | `false` | Register GORM Prometheus metrics. |
| `--skip-migrations` | mode-dependent | Skip database migrations when migrations run by default. |
| `--db-migrations` | mode-dependent | Run migrations when migrations are skipped by default. Deprecated in run-by-default mode. |
| `--otel-collector-url` | `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry gRPC collector endpoint. |
| `--otel-service-name` | caller supplied | OpenTelemetry service name. |
| `--otel-insecure` | `true` | Disable TLS for the OpenTelemetry collector. |

## Context-Aware Properties

These properties are read through `ctx.Properties()` and can be set via CLI,
file, DB, or annotations.

| Property | Type | Default | Effect |
| --- | --- | --- | --- |
| `artifacts.connection` | string | empty | Connection URL for external artifact/blob storage. Empty uses inline DB-backed blob storage. |
| `casbin.auto.save` | bool | `true` | Enables Casbin auto-save. |
| `casbin.cache` | bool | `true` | Enables the Casbin enforcer cache. |
| `casbin.cache.expiry` | duration | `1m` | Casbin cache expiry. |
| `casbin.cache.reload.interval` | duration | `5m` | Casbin policy auto-load interval. |
| `casbin.explain` | bool | `false` | Uses Casbin `EnforceEx` and logs matched rules. |
| `casbin.log.level` | int | `1` | Enables Casbin logging when `>= 2`. |
| `db.connection.timeout` | duration | `1h` | Statement timeout applied to the application DB user role. |
| `db.postgrest.timeout` | duration | `1m` | Statement timeout applied to PostgREST DB roles. |
| `envvar.cache.timeout` | duration | `5m` | Cache TTL for Kubernetes Secret and ConfigMap env var lookups. |
| `envvar.helm.cache.timeout` | duration | `envvar.cache.timeout` | Cache TTL for Helm release value lookups. |
| `envvar.lookup.timeout` | duration | `5s` | Timeout for resolving an `EnvVar` from Kubernetes sources. |
| `har.captureContentTypes` | CSV string | empty | Restricts HAR content capture to matching content types. |
| `har.maxBodySize` | int bytes | `65536` | Maximum HAR body capture size. |
| `job.ResetIsPushed.ignore_deleted_at` | bool | `false` | When true, reset-is-pushed queries do not add `deleted_at IS NULL`. |
| `job.ResetIsPushed.interval_days` | int | `7` | Lookback window for resetting `is_pushed`. |
| `job.eviction.period` | duration | `1m` | Sleep period for job-history eviction when no eviction IDs are queued. |
| `job.jitter.disable` | bool | `false` | Disables schedule jitter for periodic jobs. |
| `leader.lease.duration` | duration | `30s` | Kubernetes leader election lease duration. |
| `log.level` | string | logger default | Raises effective context observability level globally. |
| `log.level.http` | string | unset | Enables HTTP request/response header logging at `debug`; includes bodies at `trace`. |
| `log.level.http.har` | string | unset | Enables HAR capture for HTTP at `debug`; includes full body capture at `trace`. |
| `log.level.<feature>` | string | unset | Raises effective context logging for a named feature. For Kubernetes, aliases are `kubernetes`, `kubectl`, and `k8s`. |
| `log.level.<feature>.har` | string | unset | Raises effective HAR capture level for a named feature. |
| `log.level.resourceSelector` | string | unset | When set, logs generated resource selector SQL at trace level on the `resourceSelector` logger. |
| `postgres.session.<setting>` | string | unset | Applied by `ApplySessionProperties` as `SET LOCAL <setting> = '<value>'` inside a transaction. |
| `query.log` | bool | `false` | Logs resource selector and query logger output at normal verbosity. |
| `secretkeeper.cache.ttl` | duration | `10m` | TTL for the cloud secret keeper cache. |
| `shell.connection.wait_before_cleanup` | duration | `0` | Wait before cleaning up shell connection artifacts. |
| `topology.cache.age` | duration | `5m` | Cache age for topology responses. |
| `topology.query.timeout` | duration | `30s` | Default topology query timeout when the context has no deadline. |
| `update_is_pushed.batch.size` | int | `200` | Batch size for marking pushed records during upstream reconciliation. |
| `upstream.client.cache.view-columns.duration` | duration | go-cache default | Cache duration for upstream view-column client lookups. |
| `view.http.body.max_size_bytes` | int bytes | `26214400` | Maximum HTTP response body size for HTTP data queries. Non-positive values fall back to the default. |

## Mission Properties

These are additional properties used by `../incident-commander`. Unless noted
as process-global in the later table, they are resolved through
`ctx.Properties()` and can come from CLI/file, DB, or annotations.

| Property | Type | Default | Effect |
| --- | --- | --- | --- |
| `access.log` | bool | `true` | Enable access logging. |
| `access.log.colors` | bool | `true` | Enable colors in detailed access logs. |
| `access.log.debug` | bool | `false` | Enable debug logging for access log middleware. |
| `access.log.request.body` | bool | `false` | Include request bodies in access logs. |
| `access.log.request.body.max` | int bytes | `2048` | Maximum request body bytes captured by access logs. |
| `access.log.request.header` | bool | mixed defaults | Include request headers in access logs. |
| `access.log.request.id` | bool | `false` | Include request IDs in access logs. |
| `access.log.response.body` | bool | `false` | Include response bodies in access logs. |
| `access.log.response.body.max` | int bytes | `8192` | Maximum response body bytes captured by access logs. |
| `access.log.skip.sanitize` | bool | `false` | Skip access log sanitization. |
| `access.log.spanId` | bool | `true` | Include span IDs in access logs. |
| `access.log.traceId` | bool | `true` | Include trace IDs in access logs. |
| `access.log.userAgent` | bool | `false` | Include user-agent values in access logs. |
| `artifacts.max_read_size` | int bytes | `52428800` | Maximum artifact bytes read for playbook artifact responses. Values `<= 0` disable the guard. |
| `auth.impersonation` | bool/off switch | `false` | `off`, `false`, or `disabled` disables scope impersonation. |
| `dashboard.default.view` | string | `mission-control-dashboard` | Default dashboard view name or `namespace/name`. |
| `event_queue.maxAge` | duration | `720h` | Maximum age for `event_queue` rows before cleanup. |
| `events.audit.size` | int | `20` | Number of recent events retained in audit rings. |
| `<event>.batchSize` | int | handler value | Batch size for a named async event consumer, for example `notification.send.batchSize`. |
| `<event>.debug` | bool/off switch | `false` | Enables debug logging for a named async event consumer when set to `off`/`false` by current code. |
| `<event>.trace` | bool/off switch | `false` | Enables trace logging for a named async event consumer when set to `off`/`false` by current code. |
| `incidents.disable` | bool | `false` | Disable incident notification behavior. |
| `job.history.agentItemsToRetain` | int | `3` | Agent job-history entries to retain per status grouping. |
| `job.history.maxAge` | duration | `720h` | Maximum job history age before cleanup. |
| `job.history.running.maxAge` | duration | `4h` | Maximum running job age before marking stale. |
| `mcp.template.max-length` | int bytes | `65536` | Maximum MCP template size. |
| `mcp.template.timeout` | duration | `10s` | MCP template rendering timeout. |
| `metrics.agents.cache_ttl` | duration | `5m` | Prometheus agent collector cache TTL. |
| `metrics.canaries.cache_ttl` | duration | `5m` | Prometheus canary collector cache TTL. |
| `metrics.checks.cache_ttl` | duration | `5m` | Prometheus check collector cache TTL. |
| `metrics.checks.labels` | CSV string | empty | Check label include/exclude patterns for metrics. |
| `metrics.config_items.cache_ttl` | duration | `5m` | Prometheus config item collector cache TTL. |
| `metrics.disable` | CSV string | empty | Metric names to disable. `*` disables all supported metrics. |
| `metrics.prefix` | string | empty | Metric name prefix. |
| `notification.max-retries` | int | `4` | Maximum notification delivery retries. |
| `notifications.dedup.window` | duration | `24h` | Notification de-duplication window. |
| `notifications.error_reset_duration` | duration | `1h` | How long before notification errors can be reset. |
| `notifications.group_by_interval` | duration | `24h` | Default interval for grouped notifications. |
| `notifications.max.count` | int | `50` | Maximum notifications per rate-limit window. |
| `notifications.max.window` | duration | `4h` | Notification rate-limit window. |
| `playbook.action.ai.log-prompt` | bool | `false` | Log AI action prompts. |
| `playbook.action.consumers` | int | `5` | Number of playbook action consumers. |
| `playbook.consumer.timeout` | duration | `1m` | Playbook consumer timeout. |
| `playbook.retention.age` | duration | `720h` | Retention period for deleted playbooks. |
| `playbook.run.timeout` | duration | `30m` | Default playbook run timeout. |
| `playbook.runner.disabled` | bool | `false` | Disable playbook action runners. |
| `playbook.runner.longpoll.timeout` | duration | `45s` | Long-poll timeout for remote playbook runners. |
| `playbook.scheduler.disabled` | bool | `false` | Disable playbook run scheduler. |
| `playbook.schedulers` | int | `5` | Number of playbook run schedulers. |
| `rls.debug` | bool | `false` | Log RLS payloads. |
| `rls.disable` | bool | `false` | Disable RLS in startup checks. |
| `rls.enable` | bool | `false` | Enable RLS. |
| `scope.cache.ttl` | duration | `1m` | RBAC scope cache TTL. |
| `settings.user.disabled` | bool | `false` | Set by auth middleware when the current user is disabled. |
| `shorturl.defaultExpiry` | duration | `2160h` | Default short URL expiry. |
| `slack.max-url-length` | int | `50` | Maximum Slack URL length before shortening. Values above `3000` are ignored by code. |
| `upstream.pull_playbook_actions` | bool | `true` | Schedule upstream playbook action pull jobs. |
| `view.refresh.max-timeout` | duration | `1m` | Maximum timeout for asynchronous view refreshes. |

## Config DB Properties

These are additional properties used by `../config-db`.

| Property | Type | Default | Effect |
| --- | --- | --- | --- |
| `azuredevops.concurrency` | int | `5` | Azure DevOps scraper concurrency. |
| `azuredevops.pipeline.max_age` | duration | `168h` | Maximum Azure DevOps pipeline run age to scrape. |
| `azuredevops.terminal_cache.ttl` | duration | `1h` | Azure DevOps terminal status cache TTL. |
| `change_retention.delete_batch_size` | int | `1000` | Batch size for config change retention deletes. |
| `changes.dedup.disable` | bool | `false` | Disable config change de-duplication. |
| `changes.dedup.window` | duration | `1h` | Config change de-duplication window. |
| `config.retention.period` | duration | `168h` | Retention period for deleted config items. |
| `config.retention.stale_item_age` | duration | scraper default | Age after which stale config items are soft deleted. |
| `config_analysis.retention.max_age` | duration | `48h` | Age after which stale config analyses are marked resolved. |
| `config_analysis.set_status_closed_days` | int days | `7` | Days after which resolved config analyses are closed. |
| `config_scraper.retention.period` | duration | `720h` | Retention period for deleted config scraper records. |
| `diff.rust-gen` | bool | `false` | Use the alternate diff implementation when available. |
| `external.cache.timeout` | duration | `24h` | External entity cache timeout. |
| `incremental_scrape_event.lag_threshold` | duration | `30s` | Slow-event threshold for incremental scrape event logging. |
| `kubernetes.get.concurrency` | int | `10` | Concurrency for Kubernetes fetch operations. |
| `kubernetes.rbac_config_access` | bool | `true` | Enable config access generation from Kubernetes RBAC. |
| `scraper.concurrency` | int | `12` | Global config scraper concurrency. |
| `scraper.<type>.concurrency` | int | type default | Per-type scraper concurrency. Known types include `aws`, `azure`, `azuredevops`, `file`, `gcp`, `githubactions`, `http`, `kubernetes`, `kubernetesfile`, `slack`, `sql`, `terraform`, `trivy`, and `playwright`. |
| `scraper.<uid>.schedule` | string | spec/default | Per-scrape-config schedule override by UID. |
| `scraper.<type>.schedule.min` | duration | `29s` | Minimum schedule interval for a scraper type. |
| `scraper.aws.trusted_advisor.minInterval` | duration | `16h` | Minimum interval between AWS Trusted Advisor scrapes. |
| `scraper.diff.disable` | bool | `false` | Disable config diff generation. |
| `scraper.diff.timer.minSize` | int bytes | `20480` | Minimum config size before diff memory timing at high verbosity. |
| `scraper.log.items` | bool | `false` | Log config scraper item processing details. |
| `scraper.log.slow_diff_threshold` | duration | `1s` | Threshold for slow diff warnings. |
| `scraper.timeout` | duration | `4h` | Default scraper timeout. |
| `scraper.<key>` | bool | varies | `ScrapeContext.PropertyOn` prefixes keys with `scraper.` and also checks `scraper.<uid>.<key>`. Common keys include `azure.devops.incremental`, `capture.har`, `capture.logs`, `capture.snapshots`, `runNow`, `disable`, `watch.disable`, `log.exclusions`, `log.skipped`, `log.noResourceId`, `log.items`, `log.missing`, `log.relationships`, `log.rule.expr`, `log.transforms`, and `log.changes.unmatched`. |
| `scraper.scraper.label.missing` | bool | `false` | Effective key for current `PropertyOn("scraper.label.missing")` usage. |
| `scraper.scraper.tag.missing` | bool | `false` | Effective key for current `PropertyOn("scraper.tag.missing")` usage. |
| `scrapers.default.schedule` | string | startup flag default | Default schedule for scrape configs without an explicit schedule. |
| `scrapers.event.stale-timeout` | duration | `1h` | Scraper event stale timeout. |
| `scrapers.event.workers` | int | `2` | Number of scraper event workers. |
| `scrapers.githubactions.concurrency` | int | `10` | GitHub Actions API request concurrency per repository. |
| `scrapers.githubactions.maxAge` | duration | `168h` | Maximum age for GitHub Actions workflow runs. |

## Canary Checker Properties

These are additional properties used by `../canary-checker`.

| Property | Type | Default | Effect |
| --- | --- | --- | --- |
| `canary.retention.age` | duration | `168h` | Retention period for soft-deleted canaries. |
| `canary.status.max.error` | int bytes | `131072` | Maximum check status error length. |
| `canary.status.max.message` | int bytes | `4096` | Maximum check status message length. |
| `check.*.disabled` | bool | `false` | Disable canary check job synchronization. |
| `check.retention.age` | duration | `168h` | Retention period for soft-deleted checks. |
| `check.status.retention.days` | int days | `30` | Check status retention in days. |
| `checks.kubernetesResource.maxResources` | int | `10` | Maximum total Kubernetes resources allowed in a Kubernetes resource check. |
| `component.retention.period` | duration | `168h` | Retention period for soft-deleted components. |
| `components.delete_batch_size` | int | `100` | Batch size for component deletion during topology sync. |
| `http.har` | bool | `false` | Enable HAR collection. |
| `http.har.location` | string | `.` | Directory where HAR files are written. |
| `pubsub.max_messages` | int | `1000` | Maximum Pub/Sub messages read by a canary check. |
| `s3.list.max-objects` | int | `50000` | Maximum S3 objects listed by folder checks. |
| `upstream.pull_canaries` | bool | `true` | Schedule canary upstream pull jobs. |

## Flanksource UI Property Usage

`../flanksource-ui` reads `/properties` as feature flags. These are database
properties served by the API; they are not object annotations. The Settings >
Feature Flags page can create and update DB-backed rows, but rows with
`source=local` are displayed read-only.

The UI also uses many resource metadata `properties` fields, for example
connection form `properties`, topology/config display properties, and playbook
parameter UI hints such as `language`, `jsonSchemaUrl`, `options`, `filter`,
`multiline`, `min`, `max`, `minLength`, `maxLength`, and `regex`. Those are not
runtime properties and are not listed in the schema.

### UI Feature Flags

Feature flags use the property name `<feature>.disable`. A feature is disabled
only when the property value is exactly the string `true`; missing rows and any
other value leave the feature enabled.

| Property | Effect |
| --- | --- |
| `topology.disable` | Hide or disable topology UI surfaces. |
| `health.disable` | Hide or disable health UI surfaces. |
| `incidents.disable` | Hide or disable incident UI surfaces. Also disables incident notification behavior in incident-commander when read by the backend. |
| `config.disable` | Hide or disable config UI surfaces. |
| `logs.disable` | Hide or disable log UI surfaces. |
| `playbooks.disable` | Hide or disable playbook UI surfaces. |
| `applications.disable` | Hide or disable application UI surfaces. |
| `views.disable` | Hide or disable custom view UI surfaces. |
| `ai.disable` | Hide or disable AI actions and prompts in UI surfaces that check this flag. |
| `agents.disable` | Hide or disable agent UI surfaces. |
| `settings.connections.disable` | Hide or disable connection settings. |
| `settings.users.disable` | Hide or disable user settings. |
| `settings.teams.disable` | Hide or disable team settings. |
| `settings.rules.disable` | Hide or disable rules settings. |
| `settings.config_scraper.disable` | Hide or disable config scraper settings. Also disabled when `config.disable=true`. |
| `settings.topology.disable` | Hide or disable topology settings. Also disabled when `topology.disable=true`. |
| `settings.health.disable` | Hide or disable health settings. Also disabled when `health.disable=true`. |
| `settings.job_history.disable` | Hide or disable job history settings. Also disabled when `health.disable=true`. |
| `settings.feature_flags.disable` | Hide or disable the feature flags settings page. |
| `settings.logging_backends.disable` | Hide or disable logging backend settings. |
| `settings.event_queue_status.disable` | Hide or disable event queue status settings. |
| `settings.organization_profile.disable` | Hide or disable organization profile settings. |
| `settings.notifications.disable` | Hide or disable notification settings. |
| `settings.playbooks.disable` | Hide or disable playbook settings. |
| `settings.integrations.disable` | Hide or disable integration settings. |
| `settings.permissions.disable` | Hide or disable permission settings. |
| `settings.artifacts.disable` | Hide or disable artifact settings. |

### UI Snippets

| Property | Type | Source | Effect |
| --- | --- | --- | --- |
| `flanksource.ui.snippets` | JavaScript function expression string | `local` only | Executed once after the authenticated user is available. The function receives `{ user, organization }`. |

Only a `flanksource.ui.snippets` row whose `source` is `local` is executed by
the UI. DB-backed rows with the same name are fetched and visible in the
feature-flag list, but the snippet hook ignores them.

Example value:

```js
({ user, organization }) => {
  window.analytics?.identify(user?.id, {
    email: user?.email,
    organization: organization?.name
  });
}
```

### UI Dashboard Selection

| Property | Type | Effect |
| --- | --- | --- |
| `defaults.dashboard_view` | string | Used by the UI sidebar to decide which custom view should be shown as the dashboard navigation item. It accepts a view UUID, `namespace/name`, or name. |
| `dashboard.default.view` | string | Used by the backend `/api/dashboard` endpoint to resolve the actual homepage dashboard view. It accepts `namespace/name` or name and defaults to `mission-control-dashboard`. |

### UI Proxy Selection

| Property | Type | Effect |
| --- | --- | --- |
| `proxy.disable` | bool | In Clerk auth mode, overrides the organization's `direct` metadata. When true, the UI bypasses the proxy and points API clients at the organization's `backend_url`. |

### Job Properties

Jobs support global, per-job, and per-job-id property names:

| Property pattern | Type | Default | Effect |
| --- | --- | --- | --- |
| `jobs.<name>.schedule` | string | job value | Overrides a job's cron schedule. |
| `jobs.<name>.<id>.schedule` | string | job value | Intended per-id schedule override. The current lookup checks `jobs.<name>.schedule` first. |
| `jobs.<name>.timeout` | duration | job value | Overrides job timeout. |
| `jobs.<name>.<id>.timeout` | duration | job value | Intended per-id timeout override. The current lookup checks `jobs.<name>.timeout` first. |
| `jobs.<name>.history` | bool | `true` | Enables job history. |
| `jobs.<name>.<id>.history` | bool | `true` | Enables job history for a specific job id. |
| `jobs.history` | bool | `true` | Fallback for all jobs. |
| `jobs.<name>.trace` | bool | `false` | Enables trace logging for a job. |
| `jobs.<name>.<id>.trace` | bool | `false` | Enables trace logging for a specific job id. |
| `jobs.trace` | bool | `false` | Fallback for all jobs. |
| `jobs.<name>.debug` | bool | `false` | Enables debug logging for a job. |
| `jobs.<name>.<id>.debug` | bool | `false` | Enables debug logging for a specific job id. |
| `jobs.debug` | bool | `false` | Fallback for all jobs. |
| `jobs.<name>.singleton` | bool | job value | Overrides singleton behavior. |
| `jobs.<name>.<id>.singleton` | bool | job value | Overrides singleton behavior for a specific job id. |
| `jobs.singleton` | bool | job value | Fallback for all jobs. |
| `jobs.<name>.disable` | bool | `false` | Disables a job. |
| `jobs.<name>.<id>.disable` | bool | `false` | Disables a specific job id. |
| `jobs.disable` | bool | `false` | Fallback for all jobs. |
| `jobs.<name>.disabled` | bool | `false` | Alias for `disable`. |
| `jobs.<name>.<id>.disabled` | bool | `false` | Alias for `disable` for a specific job id. |
| `jobs.disabled` | bool | `false` | Fallback alias for all jobs. |
| `jobs.<name>.retention.success` | int | job value | Successful job-history entries to retain. |
| `jobs.<name>.<id>.retention.success` | int | job value | Intended per-id success retention override. The current lookup checks `jobs.<name>.retention.success` first. |
| `jobs.<name>.retention.failed` | int | job value | Failed, warning, or skipped job-history entries to retain. |
| `jobs.<name>.<id>.retention.failed` | int | job value | Intended per-id failed retention override. The current lookup checks `jobs.<name>.retention.failed` first. |

Boolean job properties are looked up from most specific to least specific:
`jobs.<name>.<id>.<key>`, `jobs.<name>.<key>`, then `jobs.<key>`.

String and int job helpers currently check `jobs.<name>.<key>` before
`jobs.<name>.<id>.<key>`.

## Process-Global Properties

These properties are read directly from the commons process-local property
store. They can be set via CLI `-P`, a loaded properties file, debug property
POSTs, or code that calls `properties.Set`, but not via DB rows or annotations.

| Property | Type | Default | Effect |
| --- | --- | --- | --- |
| `access_token.default_expiry` | duration | `2160h` | Default expiry for generated access tokens. |
| `canary.status.max.error` | int bytes | `131072` | Maximum check status error length. |
| `canary.status.max.message` | int bytes | `4096` | Maximum check status message length. |
| `change_retention.delete_batch_size` | int | `1000` | Batch size for config change retention deletes. |
| `components.delete_batch_size` | int | `100` | Batch size for component deletion during topology sync. |
| `config.traversal.cache_expiry.min` | duration | `2h` | Minimum randomized expiry for config traversal cache entries. |
| `config.traversal.cache_expiry.max` | duration | `4h` | Maximum randomized expiry for config traversal cache entries. |
| `config_analysis.set_status_closed_days` | int days | `7` | Days after which resolved config analyses are closed. |
| `db.migrate.skip` | bool | `false` | Skips database migrations in `migrate.Migrate`. |
| `diff.rust-gen` | bool | `false` | Use the alternate diff implementation when available. |
| `envvar.lookup.log` | bool | `false` | Logs resolved env var lookup values when the logger verbosity is high enough. |
| `external.cache.timeout` | duration | `24h` | External entity cache timeout in config-db. |
| `http.body.disabled` | bool | `false` | Disables HTTP body capture in commons HTTP trace middleware. |
| `http.headers.disabled` | bool | `false` | Disables HTTP header capture in commons HTTP trace middleware. |
| `http.log.response.body.length` | int bytes | `4096` | Maximum logged HTTP response body length. |
| `incremental_scrape_event.lag_threshold` | duration | `30s` | Slow-event threshold for incremental scrape event logging. |
| `job_history.agent_cleanup.batch_size` | int | `2000` | Batch size for stale agent job-history cleanup. |
| `kubernetes.cache.timeout` | duration | `240m` | Kubernetes discovery cache timeout for the REST mapper. |
| `log.color` | bool | logger flag default | Enables colored logs. Automatically forced to false when `log.json=true`. |
| `log.color.<logger>` | bool | `log.color` | Color setting for a named logger. |
| `log.caller` | bool | logger flag default | Adds source caller information to logs. |
| `log.caller.<logger>` | bool | `log.caller` | Caller setting for a named logger. |
| `db.log.level` | string | unset | Updates the `db` logger level through the commons logger property listener. |
| `log.db.maxLength` | int | `1024` | Maximum SQL log length. |
| `log.db.params` | bool | `false` | Logs SQL parameters when DB trace logging is enabled. |
| `log.db.slowThreshold` | duration | `1s` | GORM slow query threshold. |
| `log.json` | bool | logger flag default | Emits JSON logs. |
| `log.kubeproxy` | bool | `false` | Enable kube proxy logging in incident-commander. |
| `log.level` | string | `info` | Root logger level for commons loggers. |
| `log.level.<logger>` | string | `log.level` | Named logger level. Special case: `log.level.http` wraps the default HTTP client transport with a logger. |
| `log.report.caller` | bool | logger flag default | Alias used by `logger.Configure`; also updates caller reporting. |
| `log.time.format` | string | `15:04:05.000` | Log timestamp format. |
| `log.time.format.<logger>` | string | `log.time.format` | Timestamp format for a named logger. |
| `memory.stats` | duration | `0` | When positive, starts periodic memory stats logging on the debug server. |
| `metrics.auth.disabled` | bool | `false` | Disable authentication for `/metrics` in incident-commander. |
| `notification.tracing` | bool | `false` | Enable notification tracing. |
| `notifications.labels.order` | string | built-in label order | Overrides default display ordering for selected labels. |
| `notifications.labels.whitelist` | string | built-in whitelist | Overrides default label whitelist groups. |
| `pubsub.max_messages` | int | `1000` | Maximum Pub/Sub messages read by a canary check. |
| `response.strip_upstream_cors` | bool | `true` | Strip upstream CORS headers in incident-commander proxy responses. |
| `shell.allowed.envs` | CSV string | empty | Additional environment variable names passed through to shell executions. Read during package init. |
| `shell.jq.timeout` | duration | `5s` | Timeout for `jq` execution. |
| `shell.yq.timeout` | duration | `shell.jq.timeout` | Timeout for `yq` execution. |
| `smtp.debug` | bool | `false` | Enable SMTP debug logging. |
| `upstream.pull_canaries` | bool | `true` | Schedule canary upstream pull jobs. |
| `upstream.summary.fkerror_id_count` | int | `10` | Number of foreign-key error IDs included in upstream reconciliation summaries. |

## Observability Notes

HTTP and HAR context properties are feature-aware. For example:

```properties
log.level.http=debug
log.level.http.har=trace
log.level.kubernetes=debug
log.level.kubernetes.har=debug
har.maxBodySize=131072
```

For plain HTTP logging, `debug` logs headers and `trace` logs headers plus
bodies. For HAR capture, `debug` captures request/response metadata and
`trace` enables the HAR collector middleware with body capture subject to HAR
configuration.

## Discovery Endpoints

When the debug routes are registered:

| Endpoint | Result |
| --- | --- |
| `GET /debug/properties` | Supported context-aware properties that have been touched in the running process, including type, default, and current value. |
| `GET /debug/system/properties` | Process-local commons properties. |
| `POST /debug/property` | Sets a process-local commons property for the running process. |
| `echo.Properties` handler | Combined process-local and DB properties, with their source, wherever the embedding application mounts it. |

`GET /debug/properties` is populated lazily by calls to `ctx.Properties()`.
It is useful for introspection, but it is not a complete static registry until
the relevant code paths have executed.
