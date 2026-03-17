package migrate

import (
	"bufio"
	"context"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/gomplate/v3"
	"github.com/samber/lo"
	"github.com/samber/oops"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/db"
	"github.com/flanksource/duty/functions"
	premigrate "github.com/flanksource/duty/schema/pre-migrate"
	"github.com/flanksource/duty/schema"
	"github.com/flanksource/duty/views"
)

func RunMigrations(pool *sql.DB, config api.Config) error {
	l := logger.GetLogger("migrate")

	if properties.On(false, "db.migrate.skip") {
		return nil
	}

	if config.ConnectionString == "" {
		return errors.New("connection string is empty")
	}

	if pool == nil {
		return errors.New("pool is nil")
	}

	// RLS enable/disable should always be explicit
	//
	// NOTE: must always run either rls_enable or rls_disable because the properties also dictates
	// whether to run these scripts even if they haven't changed.
	if config.EnableRLS {
		config.SkipMigrationFiles = append(config.SkipMigrationFiles, "9999_rls_disable.sql")
		config.MustRun = append(config.MustRun, "9998_rls_enable.sql")
	} else if config.DisableRLS {
		config.SkipMigrationFiles = append(config.SkipMigrationFiles, "9998_rls_enable.sql")
		config.MustRun = append(config.MustRun, "9999_rls_disable.sql")
	} else {
		config.SkipMigrationFiles = append(config.SkipMigrationFiles, "9998_rls_enable.sql", "9999_rls_disable.sql")
	}

	row := pool.QueryRow("SELECT current_database();")
	var name string
	if err := row.Scan(&name); err != nil {
		return fmt.Errorf("failed to get current database: %w", err)
	}
	l.V(1).Infof("Migrating database %s", name)

	if err := createMigrationLogTable(pool); err != nil {
		return fmt.Errorf("failed to create migration log table: %w", err)
	}

	allFunctions, allPremigrations, allViews, err := GetExecutableScripts(pool, config.MustRun, config.SkipMigrationFiles)
	if err != nil {
		return fmt.Errorf("failed to get executable scripts: %w", err)
	}

	l.V(3).Infof("Running %d scripts (functions)", len(allFunctions))
	if err := runScripts(pool, allFunctions); err != nil {
		return fmt.Errorf("failed to run scripts: %w", err)
	}

	l.V(3).Infof("Granting roles to current user")
	// Grant postgrest roles in ./functions/postgrest.sql to the current user
	if err := grantPostgrestRolesToCurrentUser(pool, config); err != nil {
		return oops.Wrapf(err, "failed to grant postgrest roles")
	}

	l.V(3).Infof("Running %d scripts (pre-migrate)", len(allPremigrations))
	if err := runScripts(pool, allPremigrations); err != nil {
		return fmt.Errorf("failed to run premigration scripts: %w", err)
	}

	l.V(3).Infof("Applying schema migrations")
	if err := schema.Apply(context.TODO(), config.ConnectionString); err != nil {
		return fmt.Errorf("failed to apply schema migrations: %w", err)
	}

	l.V(3).Infof("Running %d scripts (views)", len(allViews))
	if err := runScripts(pool, allViews); err != nil {
		return fmt.Errorf("failed to run scripts for views: %w", err)
	}

	return nil
}

// GetExecutableScripts returns functions, pre-migrate scripts & views that must be applied.
// It takes dependencies into account & excludes any unchanged scripts.
func GetExecutableScripts(pool *sql.DB, mustRun, skip []string) (map[string]string, map[string]string, map[string]string, error) {
	l := logger.GetLogger("migrate")

	var (
		allFunctions      = map[string]string{}
		allPremigrations  = map[string]string{}
		allViews          = map[string]string{}
	)

	l.V(3).Infof("Getting functions")
	funcs, err := functions.GetFunctions()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get functions: %w", err)
	}

	l.V(3).Infof("Getting pre-migrate scripts")
	premigs, err := premigrate.GetPremigrations()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get premigrations: %w", err)
	}

	l.V(3).Infof("Getting views")
	views, err := views.GetViews()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get views: %w", err)
	}

	depGraph, err := getDependencyTree()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to for dependency map: %w", err)
	}

	currentMigrationHashes, err := readMigrationLogs(pool)
	if err != nil {
		return nil, nil, nil, err
	}

	env := buildMigrationEnv(pool)

	for path, content := range funcs {
		if lo.Contains(mustRun, path) {
			// proceeed. do not check hash
		} else if lo.Contains(skip, path) {
			continue
		} else if hasMatchingHash(path, content, currentMigrationHashes, env) {
			continue
		}

		allFunctions[path] = content

		for _, dependent := range depGraph[filepath.Join("functions", path)] {
			baseDir := filepath.Dir(dependent)
			filename := filepath.Base(dependent)

			switch baseDir {
			case "functions":
				allFunctions[filename] = funcs[filename]
			case "pre-migrate":
				allPremigrations[filename] = premigs[filename]
			case "views":
				allViews[filename] = views[filename]
			default:
				panic(fmt.Sprintf("unhandled base directory: %s", baseDir))
			}
		}
	}

	for path, content := range premigs {
		if lo.Contains(mustRun, path) {
			// proceeed. do not check hash
		} else if lo.Contains(skip, path) {
			continue
		} else if hasMatchingHash(path, content, currentMigrationHashes, env) {
			continue
		}

		allPremigrations[path] = content

		for _, dependent := range depGraph[filepath.Join("pre-migrate", path)] {
			baseDir := filepath.Dir(dependent)
			filename := filepath.Base(dependent)

			switch baseDir {
			case "functions":
				allFunctions[filename] = funcs[filename]
			case "pre-migrate":
				allPremigrations[filename] = premigs[filename]
			case "views":
				allViews[filename] = views[filename]
			default:
				panic(fmt.Sprintf("unhandled base directory: %s", baseDir))
			}
		}
	}

	for path, content := range views {
		if lo.Contains(mustRun, path) || isMarkedForAlwaysRun(content) {
			l.V(3).Infof("marked for always run: %s", path)
			// proceeed. do not check hash
		} else if lo.Contains(skip, path) {
			continue
		} else if hasMatchingHash(path, content, currentMigrationHashes, env) {
			continue
		}

		allViews[path] = content
		for _, dependent := range depGraph[filepath.Join("views", path)] {
			baseDir := filepath.Dir(dependent)
			filename := filepath.Base(dependent)

			switch baseDir {
			case "functions":
				allFunctions[filename] = funcs[filename]
			case "pre-migrate":
				allPremigrations[filename] = premigs[filename]
			case "views":
				allViews[filename] = views[filename]
			default:
				panic(fmt.Sprintf("unhandled base directory: %s", baseDir))
			}
		}
	}

	return allFunctions, allPremigrations, allViews, err
}

// parseHeader extracts SQL comment directives from the top of a script.
// It returns a map of directive key → value (e.g. "runs" → "always", "if" → "'table' in tables").
func parseHeader(content string) map[string]string {
	directives := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "--") {
			break
		}
		line = strings.TrimPrefix(line, "--")
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ":"); ok {
			directives[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return directives
}

func isMarkedForAlwaysRun(content string) bool {
	return parseHeader(content)["runs"] == "always"
}

func migrationLogsTableExists(pool *sql.DB) bool {
	var exists bool
	err := pool.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'migration_logs')").Scan(&exists)
	return err == nil && exists
}

func readMigrationLogs(pool *sql.DB) (map[string]string, error) {
	if !migrationLogsTableExists(pool) {
		return make(map[string]string), nil
	}

	rows, err := pool.Query("SELECT path, hash FROM migration_logs")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration logs: %w", err)
	}
	defer rows.Close()

	migrationHashes := make(map[string]string)
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, err
		}

		migrationHashes[path] = hash
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return migrationHashes, nil
}

func createRole(db *sql.DB, roleName string, config api.Config, grants ...string) error {
	if roleName == "" {
		return nil
	}
	log := logger.GetLogger("migrate")
	count := 0
	if err := db.QueryRow("SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname = $1 LIMIT 1", roleName).Scan(&count); err != nil {
		return err
	} else if count == 0 {
		if _, err := db.Exec(fmt.Sprintf("CREATE ROLE %s", roleName)); err != nil {
			return err
		} else {
			log.Infof("Created role %s", roleName)
		}
	}
	user := config.GetUsername()
	if user == "" {
		log.Errorf("Unable to find current user, %s may not be setup correctly", roleName)
	} else {
		if granted, err := checkIfRoleIsGranted(db, roleName, user); err != nil {
			return err
		} else if !granted {
			if _, err := db.Exec(fmt.Sprintf(`GRANT %s TO "%s"`, roleName, user)); err != nil {
				log.Errorf("Failed to grant role %s to %s", roleName, user)
			} else {
				log.Infof("Granted %s to %s", roleName, user)
			}
		}
	}

	for _, grant := range grants {
		if _, err := db.Exec(fmt.Sprintf(grant, roleName)); err != nil {
			log.Errorf("Failed to apply grant[%s] for %s: %+v", grant, roleName, err)
		}
	}

	return nil
}

func grantPostgrestRolesToCurrentUser(pool *sql.DB, config api.Config) error {
	if err := createRole(pool, config.Postgrest.DBRole, config,
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s",
		"GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s",
		"GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO %s",
		"ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO %s"); err != nil {
		return err
	}
	if err := createRole(pool, config.Postgrest.AnonDBRole, config,
		"GRANT SELECT ON ALL TABLES IN SCHEMA public TO %s",
		"ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO %s"); err != nil {
		return err
	}

	return nil
}

func checkIfRoleIsGranted(pool *sql.DB, group, member string) (bool, error) {
	query := `
        SELECT COUNT(*)
        FROM pg_auth_members a, pg_roles b, pg_roles c
        WHERE
            a.roleid = b.oid AND
            a.member = c.oid AND b.rolname = $1 AND c.rolname = $2
    `

	row := pool.QueryRow(query, group, member)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	return true, nil
}

func queryTableNames(pool *sql.DB) []string {
	rows, err := pool.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	return tables
}

func buildMigrationEnv(pool *sql.DB) map[string]any {
	return map[string]any{
		"tables":     queryTableNames(pool),
		"properties": properties.Global.GetAll(),
	}
}

func runScripts(pool *sql.DB, scripts map[string]string) error {
	l := logger.GetLogger("migrate")

	var filenames []string
	for name := range scripts {
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)

	env := buildMigrationEnv(pool)

	for _, file := range filenames {
		content, ok := scripts[file]
		if !ok {
			continue
		}

		header := parseHeader(content)
		hash := computeHash(content, env)

		if expr := header["if"]; expr != "" {
			ok, err := gomplate.RunTemplateBool(env, gomplate.Template{Expression: expr})
			if err != nil {
				return fmt.Errorf("failed to evaluate if expression for %s: %w", file, err)
			}
			if !ok {
				l.V(3).Infof("skipping %s: condition not met: %s", file, expr)
				if _, err := pool.Exec("INSERT INTO migration_logs(path, hash) VALUES($1, $2) ON CONFLICT (path) DO UPDATE SET hash = $2", file, hash[:]); err != nil {
					return fmt.Errorf("failed to save migration log %s: %w", file, err)
				}
				continue
			}
		}

		l.Tracef("running script %s", file)
		if _, err := pool.Exec(scripts[file]); err != nil {
			return fmt.Errorf("failed to run script %s: %w", file, db.ErrorDetails(err))
		}

		if _, err := pool.Exec("INSERT INTO migration_logs(path, hash) VALUES($1, $2) ON CONFLICT (path) DO UPDATE SET hash = $2", file, hash[:]); err != nil {
			return fmt.Errorf("failed to save migration log %s: %w", file, err)
		}
	}

	return nil
}

func createMigrationLogTable(pool *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS migration_logs (
		path VARCHAR(255) NOT NULL,
		hash bytea NOT NULL,
		updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
		PRIMARY KEY (path)
	)`
	_, err := pool.Exec(query)
	return err
}

// computeHash returns a SHA1 hash of the script content plus resolved directive values.
// Including directive state in the hash ensures that when a CEL condition
// changes, the hash mismatches and the script is re-evaluated.
func computeHash(content string, env map[string]any) [20]byte {
	header := parseHeader(content)
	extra := ""
	if expr := header["if"]; expr != "" {
		result, _ := gomplate.RunTemplateBool(env, gomplate.Template{Expression: expr})
		extra += fmt.Sprintf("\n--if:%s=%v", expr, result)
	}
	return sha1.Sum([]byte(content + extra))
}

func hasMatchingHash(path, content string, currentHashes map[string]string, env map[string]any) bool {
	hash := computeHash(content, env)
	currentHash, exists := currentHashes[path]
	return exists && currentHash == string(hash[:])
}
