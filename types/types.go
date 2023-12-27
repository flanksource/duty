package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SQLServerType = "sqlserver"
	PostgresType  = "postgres"
	SqliteType    = "sqlite"
	MysqlType     = "mysql"
	TextType      = "TEXT"
	JSONType      = "JSON"
	JSONBType     = "JSONB"
	NVarcharType  = "NVARCHAR(MAX)"
)

const PostgresTimestampFormat = "2006-01-02T15:04:05.999999"

// NullString sets null in database on save for empty strings
type NullString sql.NullString

// Scan implements the Scanner interface.
func (s *NullString) Scan(value any) error {
	if value == nil {
		s.String, s.Valid = "", false
		return nil
	}
	s.Valid = true
	s.String = fmt.Sprint(value)
	return nil
}

// Value implements the driver Valuer interface.
func (s NullString) Value() (driver.Value, error) {
	if !s.Valid {
		return nil, nil
	}
	return s.String, nil
}

// MarshalJSON to output non base64 encoded []byte
func (s NullString) MarshalJSON() ([]byte, error) {
	if !s.Valid {
		return []byte("null"), nil
	}
	if s.String == "\"\"" {
		return []byte(""), nil
	}
	return json.Marshal(s.String)
}

// UnmarshalJSON to deserialize []byte
func (s *NullString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || string(b) == "" {
		*s = NullString{
			Valid: false,
		}
		return nil
	}

	var val string
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	*s = NullString{
		String: val,
		Valid:  true,
	}
	return nil
}

// GenericStructValue can be set as the Value() func for any json struct
func GenericStructValue[T any](t T, defaultNull bool) (driver.Value, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return b, err
	}
	if defaultNull && string(b) == "{}" {
		return nil, nil
	}
	return string(b), nil
}

// GenericStructScan can be set as the Scan(val) func for any json struct
func GenericStructScan[T any](t *T, val any) error {
	if val == nil {
		t = new(T)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal JSONB value: %v", val)
	}
	err := json.Unmarshal(ba, &t)
	return err
}

func GormValue(t any) clause.Expr {
	data, _ := json.Marshal(t)
	if string(data) == "null" {
		return gorm.Expr("NULL")
	}

	return gorm.Expr("?", string(data))
}
