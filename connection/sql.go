package connection

import (
	databasesql "database/sql"
	"fmt"
	"slices"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var supportedSQLTypes = []string{
	models.ConnectionTypePostgres,
	models.ConnectionTypeMySQL,
	models.ConnectionTypeSQLServer,
}

// +kubebuilder:object:generate=true
type SQLConnection struct {
	ConnectionName string       `yaml:"connection,omitempty" json:"connection,omitempty"`
	Type           string       `yaml:"type,omitempty" json:"type,omitempty"`
	URL            types.EnvVar `yaml:"url,omitempty" json:"url,omitempty"`
	Username       types.EnvVar `yaml:"username,omitempty" json:"username,omitempty"`
	Password       types.EnvVar `yaml:"password,omitempty" json:"password,omitempty"`

	client *databasesql.DB
}

func (s *SQLConnection) FromModel(connection models.Connection) error {
	if !isSupportedSQLType(connection.Type) {
		return fmt.Errorf("connection of type %s cannot be used with sql, expected one of %s", connection.Type, strings.Join(supportedSQLTypes, ", "))
	}

	s.ConnectionName = connection.Name
	s.Type = connection.Type
	s.URL = types.EnvVar{ValueStatic: connection.URL}
	s.Username = types.EnvVar{ValueStatic: connection.Username}
	s.Password = types.EnvVar{ValueStatic: connection.Password}
	return nil
}

func (s SQLConnection) ToModel() models.Connection {
	connType := s.Type
	if connType == "" {
		connType = models.ConnectionTypePostgres
	}

	conn := models.Connection{
		Name:     s.ConnectionName,
		Type:     connType,
		URL:      s.URL.ValueStatic,
		Username: s.Username.ValueStatic,
		Password: s.Password.ValueStatic,
	}

	return conn
}

// Client creates and returns a database/sql DB client
//
// NOTE: Must be run on a hydrated SQLConnection.
func (s *SQLConnection) Client(ctx context.Context) (*databasesql.DB, error) {
	if s.client != nil {
		return s.client, nil
	}

	if s.Type == "" {
		s.Type = models.ConnectionTypePostgres
	}

	driverName, err := sqlDriverName(s.Type)
	if err != nil {
		return nil, err
	}

	if s.URL.ValueStatic == "" {
		return nil, fmt.Errorf("sql connection url cannot be empty")
	}

	client, err := databasesql.Open(driverName, s.URL.ValueStatic)
	if err != nil {
		return nil, err
	}

	s.client = client
	return s.client, nil
}

func (s *SQLConnection) Close() error {
	if s.client == nil {
		return nil
	}

	err := s.client.Close()
	s.client = nil
	return err
}

func (s *SQLConnection) HydrateConnection(ctx context.Context) error {
	if s.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(s.ConnectionName)
		if err != nil {
			return fmt.Errorf("could not hydrate connection[%s]: %w", s.ConnectionName, err)
		}
		if connection == nil {
			return fmt.Errorf("connection[%s] not found", s.ConnectionName)
		}
		if err := s.FromModel(*connection); err != nil {
			return err
		}
	}

	ns := ctx.GetNamespace()

	if v, err := ctx.GetEnvValueFromCache(s.URL, ns); err != nil {
		return fmt.Errorf("could not get sql url from env var: %w", err)
	} else {
		s.URL.ValueStatic = v
	}

	if v, err := ctx.GetEnvValueFromCache(s.Username, ns); err != nil {
		return fmt.Errorf("could not get sql username from env var: %w", err)
	} else {
		s.Username.ValueStatic = v
	}

	if v, err := ctx.GetEnvValueFromCache(s.Password, ns); err != nil {
		return fmt.Errorf("could not get sql password from env var: %w", err)
	} else {
		s.Password.ValueStatic = v
	}

	return nil
}

func sqlDriverName(connectionType string) (string, error) {
	switch connectionType {
	case models.ConnectionTypePostgres:
		return "pgx", nil
	case models.ConnectionTypeMySQL:
		return "mysql", nil
	case models.ConnectionTypeSQLServer:
		return "sqlserver", nil
	default:
		return "", fmt.Errorf("unsupported sql connection type: %s", connectionType)
	}
}

func isSupportedSQLType(connectionType string) bool {
	return slices.Contains(supportedSQLTypes, connectionType)
}
