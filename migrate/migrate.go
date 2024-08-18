package migrate

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/db"
	"github.com/flanksource/duty/functions"
	"github.com/flanksource/duty/schema"
	"github.com/flanksource/duty/views"
)

type MigrateOptions struct {
	Skip        bool // Skip running migrations
	IgnoreFiles []string
}

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

	row := pool.QueryRow("SELECT current_database();")
	var name string
	if err := row.Scan(&name); err != nil {
		return fmt.Errorf("failed to get current database: %w", err)
	}
	l.Infof("Migrating database %s", name)

	if err := createMigrationLogTable(pool); err != nil {
		return fmt.Errorf("failed to create migration log table: %w", err)
	}

	l.V(3).Infof("Getting functions")
	funcs, err := functions.GetFunctions()
	if err != nil {
		return fmt.Errorf("failed to get functions: %w", err)
	}

	l.V(3).Infof("Running scripts")
	if err := runScripts(pool, funcs, config.SkipMigrationFiles); err != nil {
		return fmt.Errorf("failed to run scripts: %w", err)
	}

	l.V(3).Infof("Granting roles to current user")
	// Grant postgrest roles in ./functions/postgrest.sql to the current user
	if err := grantPostgrestRolesToCurrentUser(pool, config); err != nil {
		return fmt.Errorf("failed to grant postgrest roles: %w", err)
	}

	l.V(3).Infof("Applying schema migrations")
	if err := schema.Apply(context.TODO(), config.ConnectionString); err != nil {
		return fmt.Errorf("failed to apply schema migrations: %w", err)
	}

	l.V(3).Infof("Getting views")
	views, err := views.GetViews()
	if err != nil {
		return fmt.Errorf("failed to get views: %w", err)
	}

	l.V(3).Infof("Running scripts for views")
	if err := runScripts(pool, views, config.SkipMigrationFiles); err != nil {
		return fmt.Errorf("failed to run scripts for views: %w", err)
	}

	return nil
}

func grantPostgrestRolesToCurrentUser(pool *sql.DB, config api.Config) error {
	l := logger.GetLogger("migrate")

	user := config.GetUsername()
	if user == "" {
		return fmt.Errorf("Cannot find username in connection string")
	}
	role := config.Postgrest.DBRole
	if isPostgrestAPIGranted, err := checkIfRoleIsGranted(pool, role, user); err != nil {
		return err
	} else if !isPostgrestAPIGranted {
		if _, err := pool.Exec(fmt.Sprintf(`GRANT %s TO "%s"`, role, user)); err != nil {
			return err
		}
		l.Debugf("Granted %s to %s", role, user)

		grantQuery := `
				ALTER ROLE %s SET statement_timeout = '120s';
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s;
				GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s;
				GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO %s;
				ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO %s;
        `
		if _, err := pool.Exec(fmt.Sprintf(grantQuery, role, role, role, role, role)); err != nil {
			return err
		}
		l.Debugf("Granted privileges to %s", user)

	}

	isPostgrestAnonGranted, err := checkIfRoleIsGranted(pool, config.Postgrest.DBAnonRole, user)
	if err != nil {
		return err
	}
	if !isPostgrestAnonGranted {
		if _, err := pool.Exec(fmt.Sprintf(`GRANT %s TO "%s"`, config.Postgrest.DBAnonRole, user)); err != nil {
			return err
		}
		l.Debugf("Granted %s to %s", config.Postgrest.DBAnonRole, user)
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

func runScripts(pool *sql.DB, scripts map[string]string, ignoreFiles []string) error {
	l := logger.GetLogger("migrate")
	var filenames []string
	for name := range scripts {
		if collections.Contains(ignoreFiles, name) {
			continue
		}
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)

	for _, file := range filenames {
		content, ok := scripts[file]
		if !ok {
			continue
		}

		var currentHash string
		if err := pool.QueryRow("SELECT hash FROM migration_logs WHERE path = $1", file).Scan(&currentHash); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		hash := sha1.Sum([]byte(content))
		if string(hash[:]) == currentHash {
			l.V(3).Infof("Skipping script %s", file)
			continue
		}

		l.Tracef("Running script %s", file)
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
