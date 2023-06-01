package migrate

import (
	"context"
	"database/sql"
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
