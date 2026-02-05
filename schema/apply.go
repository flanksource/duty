// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found at
// https://github.com/ariga/atlas/blob/master/LICENSE

package schema

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"

	"ariga.io/atlas/sql/migrate"
	_ "ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/flanksource/commons/logger"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type stateReadCloser struct {
	migrate.StateReader
	io.Closer        // optional close function
	schema    string // in case we work on a single schema
	hcl       bool   // true if state was read from HCL files since in that case we always compare realms
}

//go:embed *.hcl
var schemas embed.FS

func skipDropTables(changes []schema.Change) []schema.Change {
	var filtered []schema.Change
	for _, change := range changes {
		switch change := change.(type) {
		case *schema.DropTable:
			logger.GetLogger("migrate").Tracef("Skipping drop table of %s", change.T.Name)
		default:
			filtered = append(filtered, change)
		}
	}
	return filtered
}

func Apply(ctx context.Context, connection string) error {
	log := logger.GetLogger("migrate")

	// https://atlasgo.io/versioned/diff#exclude-objects
	exclude := []string{
		"config_items.properties_values",
		"components.properties_values",
		"config_locations.config_locations_location_pattern_idx",

		// These indexes are managed in the views/037_notification_group_resources.sql file
		// as they are dependent on the PostgreSQL version.
		"notification_group_resources.unique_notification_group_resources_unresolved",
		"notification_group_resources.unique_notification_group_resources_unresolved_config",
		"notification_group_resources.unique_notification_group_resources_unresolved_check",
		"notification_group_resources.unique_notification_group_resources_unresolved_component",
	}

	from, err := dbReader(ctx, connection, exclude)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer from.Close()

	client, ok := from.Closer.(*sqlclient.Client)
	if !ok {
		return errors.New("--url must be a database connection")
	}

	to, err := hclStateReader(ctx, client, schemas)
	if err != nil {
		return fmt.Errorf("failed to initiate HCL state reader: %w", err)
	}
	defer to.Close()

	changes, err := computeDiff(ctx, client, from, to)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}

	if len(changes) == 0 {
		log.Debugf("No changes detected")
		return nil
	}

	changes = skipDropTables(changes)

	var plan *migrate.Plan
	if plan, err = client.PlanChanges(ctx, "", changes); err != nil {
		return fmt.Errorf("failed to plan changes: %w", err)
	}

	for _, change := range plan.Changes {
		log.Tracef("%s", change.Cmd)
	}

	if err = client.ApplyChanges(ctx, changes); err != nil {
		return fmt.Errorf("applied %d changes and then failed: %w", len(changes), err)
	}

	log.V(1).Infof("Applied %d changes", len(changes))
	return nil
}

func computeDiff(ctx context.Context, differ *sqlclient.Client, from, to *stateReadCloser) ([]schema.Change, error) {
	current, err := from.ReadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	var diff []schema.Change
	switch {
	// Compare realm if the desired state is an HCL file or both connections are not bound to a schema.
	case from.hcl, to.hcl, from.schema == "" && to.schema == "":
		diff, err = differ.RealmDiff(current, desired)
		if err != nil {
			return nil, fmt.Errorf("failed to diff realms: %w", err)
		}
	case from.schema == "", to.schema == "":
		return nil, fmt.Errorf("cannot diff a schema with a database connection: %q <> %q", from.schema, to.schema)
	default:
		// SchemaDiff checks for name equality which is irrelevant in the case
		// the user wants to compare their contents, reset them to allow the comparison.
		current.Schemas[0].Name, desired.Schemas[0].Name = "", ""
		diff, err = differ.SchemaDiff(current.Schemas[0], desired.Schemas[0])
		if err != nil {
			return nil, fmt.Errorf("failed to diff schemas: %w", err)
		}
	}
	return diff, nil
}

// hclStateReadr returns a StateReader that reads the state from the given HCL paths urls.
func hclStateReader(ctx context.Context, client *sqlclient.Client, fs embed.FS) (*stateReadCloser, error) {
	scripts, err := schemas.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read scripts: %w", err)
	}

	p := hclparse.NewParser()

	for _, file := range scripts {
		script, err := schemas.ReadFile(file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read script %s: %w", file.Name(), err)
		}

		_, diag := p.ParseHCL(script, file.Name())
		if diag.HasErrors() {
			return nil, diag
		}
	}

	realm := &schema.Realm{}
	if err := client.Eval(p, realm, make(map[string]cty.Value)); err != nil {
		return nil, fmt.Errorf("failed to evaluate HCL: %w", err)
	}

	t := &stateReadCloser{StateReader: migrate.Realm(realm), hcl: true}
	return t, nil
}

func dbReader(ctx context.Context, connection string, exclude []string) (*stateReadCloser, error) {
	c, err := sqlclient.Open(ctx, connection)
	if err != nil {
		return nil, err
	}
	sr := migrate.SchemaConn(c.Driver, c.URL.Schema, &schema.InspectOptions{Exclude: exclude})

	return &stateReadCloser{
		StateReader: sr,
		Closer:      c,
		schema:      c.URL.Schema,
	}, nil
}

// Close redirects calls to Close to the enclosed io.Closer.
func (sr *stateReadCloser) Close() {
	if sr.Closer != nil {
		sr.Closer.Close()
	}
}
