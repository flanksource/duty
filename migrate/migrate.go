package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sort"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/functions"
	"github.com/flanksource/duty/schema"
	"github.com/flanksource/duty/views"
	"github.com/pkg/errors"
)

type MigrateOptions struct {
	IgnoreFiles []string
}

func RunMigrations(pool *sql.DB, connection string, opts MigrateOptions) error {
	if connection == "" {
		return errors.New("connection string is empty")
	}
	if pool == nil {
		return errors.New("pool is nil")
	}

	row := pool.QueryRow("SELECT current_database();")
	var name string
	if err := row.Scan(&name); err != nil {
		return errors.Wrap(err, "failed to get current database")
	}
	logger.Infof("Migrating database %s", name)

	funcs, err := functions.GetFunctions()
	if err != nil {
		return err
	}

	if err := runScripts(pool, funcs, opts.IgnoreFiles); err != nil {
		return err
	}

	// Grant postgrest roles in ./functions/postgrest.sql to the current user
	if err := grantPostgrestRolesToCurrentUser(pool, connection); err != nil {
		return err
	}

	logger.Debugf("Applying schema migrations")
	if err := schema.Apply(context.TODO(), connection); err != nil {
		return err
	}

	views, err := views.GetViews()
	if err != nil {
		return err
	}

	if err := runScripts(pool, views, opts.IgnoreFiles); err != nil {
		return err
	}

	return nil
}

func grantPostgrestRolesToCurrentUser(pool *sql.DB, connection string) error {
	parsedConn, err := url.Parse(connection)
	if err != nil {
		return err
	}
	user := parsedConn.User.Username()
	if user == "" {
		return fmt.Errorf("malformed connection string, got empty username: %s", parsedConn.Redacted())
	}

	isPostgrestAPIGranted, err := checkIfRoleIsGranted(pool, "postgrest_api", user)
	if err != nil {
		return err
	}
	if !isPostgrestAPIGranted {
		if _, err := pool.Exec(fmt.Sprintf(`GRANT postgrest_api TO %s`, user)); err != nil {
			return err
		}
		logger.Debugf("Granted postgrest_api to %s", user)
	}

	isPostgrestAnonGranted, err := checkIfRoleIsGranted(pool, "postgrest_anon", user)
	if err != nil {
		return err
	}
	if !isPostgrestAnonGranted {
		if _, err := pool.Exec(fmt.Sprintf(`GRANT postgrest_anon TO %s`, user)); err != nil {
			return err
		}
		logger.Debugf("Granted postgrest_anon to %s", user)
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
	var filenames []string
	for name := range scripts {
		if collections.Contains(ignoreFiles, name) {
			continue
		}
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)
	for _, file := range filenames {
		logger.Tracef("Running script %s", file)
		if _, err := pool.Exec(scripts[file]); err != nil {
			return errors.Wrapf(err, "failed to run script %s", file)
		}
	}
	return nil
}
