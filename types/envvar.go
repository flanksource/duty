package types

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const EnvVarType = "env_var"

// +kubebuilder:object:generate=true
type EnvVar struct {
	Name        string        `json:"name,omitempty" yaml:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	ValueStatic string        `json:"value,omitempty" yaml:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	ValueFrom   *EnvVarSource `json:"valueFrom,omitempty" yaml:"valueFrom,omitempty" protobuf:"bytes,3,opt,name=valueFrom"`
}

// With* interfaces provide a mechanism for connections to have their values overwritten by values specified lower in the
// heirachy

type WithUsernamePassword interface {
	GetUsername() EnvVar
	GetPassword() EnvVar
}

type WithCertificate interface {
	GetCertificate() EnvVar
}

type WithURL interface {
	GetURL() EnvVar
}

type WithProperties interface {
	GetProperties() map[string]string
}

type GetEnvVarFromCache interface {
	GetEnvValueFromCache(e EnvVar, namespace string) (string, error)
}

func (e EnvVar) String() string {
	if e.ValueFrom == nil {
		return e.ValueStatic
	}
	return e.ValueFrom.String()
}

func (e EnvVar) IsEmpty() bool {
	return e.ValueStatic == "" && e.ValueFrom == nil
}

// +kubebuilder:object:generate=true
type EnvVarSource struct {
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty" yaml:"configMapKeyRef,omitempty" protobuf:"bytes,1,opt,name=configMapKeyRef"`
	SecretKeyRef    *SecretKeySelector    `json:"secretKeyRef,omitempty" yaml:"secretKeyRef,omitempty" protobuf:"bytes,2,opt,name=secretKeyRef"`
}

func (e EnvVarSource) String() string {
	if e.ConfigMapKeyRef != nil {
		return "configmap://" + e.ConfigMapKeyRef.String()
	}
	if e.SecretKeyRef != nil {
		return "secret://" + e.SecretKeyRef.String()
	}
	return ""
}

// +kubebuilder:object:generate=true
type ConfigMapKeySelector struct {
	LocalObjectReference `json:",inline" yaml:",inline" protobuf:"bytes,1,opt,name=localObjectReference"`
	Key                  string `json:"key" yaml:"key" protobuf:"bytes,2,opt,name=key"`
}

func (c ConfigMapKeySelector) String() string {
	return c.Name + "/" + c.Key
}

// +kubebuilder:object:generate=true
type SecretKeySelector struct {
	LocalObjectReference `json:",inline" yaml:",inline" protobuf:"bytes,1,opt,name=localObjectReference"`
	Key                  string `json:"key" yaml:"key" protobuf:"bytes,2,opt,name=key"`
}

func (s SecretKeySelector) String() string {
	return s.Name + "/" + s.Key
}

// +kubebuilder:object:generate=true
type LocalObjectReference struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
}

// GormDataType gorm common data type
func (EnvVar) GormDataType() string {
	return EnvVarType
}

// GormDBDataType gorm db data type
func (EnvVar) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case SqliteType:
		return Text
	case MysqlType:
		return Text
	case PostgresType:
		return Text
	}

	return ""
}

// Value return string value, implement driver.Valuer interface
func (e EnvVar) Value() (driver.Value, error) {
	if e.ValueFrom == nil {
		return e.ValueStatic, nil
	}
	return e.ValueFrom.String(), nil
}

// Scan scan value into string, implements sql.Scanner interface
func (e *EnvVar) Scan(value any) error {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "configmap://") {
			if len(strings.Split(v, "/")) != 4 {
				return fmt.Errorf("invalid configmap reference: %s", v)
			}
			*e = EnvVar{
				ValueFrom: &EnvVarSource{
					ConfigMapKeyRef: &ConfigMapKeySelector{
						LocalObjectReference: LocalObjectReference{
							Name: strings.Split(v, "/")[2],
						},
						Key: strings.Split(v, "/")[3],
					},
				}}
			return nil
		}
		if strings.HasPrefix(v, "secret://") {
			if len(strings.Split(v, "/")) != 4 {
				return fmt.Errorf("invalid secret reference: %s", v)
			}
			*e = EnvVar{
				ValueFrom: &EnvVarSource{
					SecretKeyRef: &SecretKeySelector{
						LocalObjectReference: LocalObjectReference{
							Name: strings.Split(v, "/")[2],
						},
						Key: strings.Split(v, "/")[3],
					},
				}}
			return nil
		}
		*e = EnvVar{
			ValueStatic: v,
		}
		return nil
	default:
		return fmt.Errorf("invalid value type: %T", value)
	}
}
