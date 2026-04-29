package api

const (
	PropertyArtifactsConnection = "artifacts.connection"

	PropertyCasbinAutoSave            = "casbin.auto.save"
	PropertyCasbinCache               = "casbin.cache"
	PropertyCasbinCacheExpiry         = "casbin.cache.expiry"
	PropertyCasbinCacheReloadInterval = "casbin.cache.reload.interval"
	PropertyCasbinExplain             = "casbin.explain"
	PropertyCasbinLogLevel            = "casbin.log.level"

	PropertyConfigTraversalCacheExpiryMax = "config.traversal.cache_expiry.max"
	PropertyConfigTraversalCacheExpiryMin = "config.traversal.cache_expiry.min"

	PropertyDBConnectionTimeout = "db.connection.timeout"
	PropertyDBMigrateSkip       = "db.migrate.skip"
	PropertyDBPostgrestTimeout  = "db.postgrest.timeout"

	PropertyEnvvarCacheTimeout     = "envvar.cache.timeout"
	PropertyEnvvarHelmCacheTimeout = "envvar.helm.cache.timeout"
	PropertyEnvvarLookupLog        = "envvar.lookup.log"
	PropertyEnvvarLookupTimeout    = "envvar.lookup.timeout"

	PropertyJobEvictionPeriod               = "job.eviction.period"
	PropertyJobHistoryAgentCleanupBatchSize = "job_history.agent_cleanup.batch_size"
	PropertyJobJitterDisable                = "job.jitter.disable"
	PropertyJobResetIsPushedIgnoreDeletedAt = "job.ResetIsPushed.ignore_deleted_at"
	PropertyJobResetIsPushedIntervalDays    = "job.ResetIsPushed.interval_days"

	PropertyKubernetesCacheTimeout = "kubernetes.cache.timeout"

	PropertyLeaderLeaseDuration = "leader.lease.duration"

	PropertyLogDBMaxLength           = "log.db.maxLength"
	PropertyLogDBParams              = "log.db.params"
	PropertyLogDBSlowThreshold       = "log.db.slowThreshold"
	PropertyLogLevelResourceSelector = "log.level.resourceSelector"

	PropertyMemoryStats = "memory.stats"

	PropertyQueryLog = "query.log"

	PropertySecretKeeperCacheTTL = "secretkeeper.cache.ttl"

	PropertyShellAllowedEnvs                 = "shell.allowed.envs"
	PropertyShellConnectionWaitBeforeCleanup = "shell.connection.wait_before_cleanup"
	PropertyShellJQTimeout                   = "shell.jq.timeout"
	PropertyShellYQTimeout                   = "shell.yq.timeout"

	PropertyTopologyCacheAge     = "topology.cache.age"
	PropertyTopologyQueryTimeout = "topology.query.timeout"

	PropertyUpdateIsPushedBatchSize = "update_is_pushed.batch.size"

	PropertyUpstreamClientCacheViewColumns = "upstream.client.cache.view-columns.duration"
	PropertyUpstreamSummaryFKErrorIDCount  = "upstream.summary.fkerror_id_count"

	PropertyViewHTTPBodyMaxSizeBytes = "view.http.body.max_size_bytes"
)

const (
	PropertyJobsPrefix = "jobs."
)
