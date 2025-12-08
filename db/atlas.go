package db

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

// NewAtlasClient returns an atlas client backed by the current connection pool.
func NewAtlasClient(ctx context.Context, pool schema.ExecQuerier) (*sqlclient.Client, func() error, error) {
	drv, err := postgres.Open(pool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a migration driver for atlas: %w", err)
	}

	return &sqlclient.Client{
		Name:   postgres.DriverName,
		Driver: drv,
	}, nil, err
}
