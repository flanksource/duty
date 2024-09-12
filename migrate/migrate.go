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
	"github.com/samber/oops"
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
		return oops.Wrapf(err, "failed to grant postgrest roles")
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
			log.Errorf("Failed to apply grants for %s: %+v", roleName, err)
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
