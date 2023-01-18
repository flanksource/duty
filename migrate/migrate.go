package migrate

import (
	"context"
	"database/sql"
	"sort"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/functions"
	"github.com/flanksource/duty/schema"
	"github.com/flanksource/duty/views"
	"github.com/pkg/errors"
)

func Migrate(Pool *sql.DB, connection string) error {
	if connection == "" {
		return errors.New("connection string is empty")
	}
	if Pool == nil {
		return errors.New("pool is nil")
	}

	row := Pool.QueryRow("SELECT current_database();")
	var name string
	if err := row.Scan(&name); err != nil {
		return errors.Wrap(err, "failed to get current database")
	}
	logger.Infof("Migrating database %s", name)

	funcs, err := functions.GetFunctions()
	if err != nil {
		return err
	}

	if err := runScripts(Pool, funcs); err != nil {
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

	if err := runScripts(Pool, views); err != nil {
		return err
	}

	return nil
}

func runScripts(Pool *sql.DB, scripts map[string]string) error {
	var filenames []string
	for name := range scripts {
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)
	for _, file := range filenames {
		logger.Debugf("Running script %s", file)
		if _, err := Pool.Exec(scripts[file]); err != nil {
			return errors.Wrapf(err, "failed to run script %s", file)
		}
	}
	return nil
}
