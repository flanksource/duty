package types

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/flanksource/commons/collections"
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
	return e.ValueStatic == "" && (e.ValueFrom == nil || e.ValueFrom.IsEmpty())
}

// +kubebuilder:object:generate=true
type EnvVarSource struct {
	// ServiceAccount specifies the service account whose token should be fetched
	ServiceAccount  *string               `json:"serviceAccount,omitempty" yaml:"serviceAccount,omitempty" protobuf:"bytes,1,opt,name=serviceAccount"`
	HelmRef         *HelmRefKeySelector   `json:"helmRef,omitempty" yaml:"helmRef,omitempty" protobuf:"bytes,2,opt,name=helmRef"`
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty" yaml:"configMapKeyRef,omitempty" protobuf:"bytes,3,opt,name=configMapKeyRef"`
	SecretKeyRef    *SecretKeySelector    `json:"secretKeyRef,omitempty" yaml:"secretKeyRef,omitempty" protobuf:"bytes,4,opt,name=secretKeyRef"`
}

func (e EnvVarSource) IsEmpty() bool {
	return (e.ServiceAccount == nil || *e.ServiceAccount == "") &&
		(e.HelmRef == nil || e.HelmRef.IsEmpty()) &&
		(e.ConfigMapKeyRef == nil || e.ConfigMapKeyRef.IsEmpty()) &&
		(e.SecretKeyRef == nil || e.SecretKeyRef.IsEmpty())
}

func (e EnvVarSource) String() string {
	if e.ConfigMapKeyRef != nil {
		return "configmap://" + e.ConfigMapKeyRef.String()
	}
	if e.SecretKeyRef != nil {
		return "secret://" + e.SecretKeyRef.String()
	}
	if e.ServiceAccount != nil {
		return "serviceaccount://" + *e.ServiceAccount
	}
	if e.HelmRef != nil {
		return "helm://" + e.HelmRef.String()
	}
	return ""
}

// +kubebuilder:object:generate=true
type HelmRefKeySelector struct {
	LocalObjectReference `json:",inline" yaml:",inline" protobuf:"bytes,1,opt,name=localObjectReference"`
	// Key is a JSONPath expression used to fetch the key from the merged JSON.
	Key string `json:"key" yaml:"key" protobuf:"bytes,2,opt,name=key"`
}

func (e HelmRefKeySelector) IsEmpty() bool {
	return e.Key == ""
}

func (c HelmRefKeySelector) String() string {
	return c.Name + "/" + c.Key
}

// +kubebuilder:object:generate=true
type ConfigMapKeySelector struct {
	LocalObjectReference `json:",inline" yaml:",inline" protobuf:"bytes,1,opt,name=localObjectReference"`
	Key                  string `json:"key" yaml:"key" protobuf:"bytes,2,opt,name=key"`
}

func (c ConfigMapKeySelector) IsEmpty() bool {
	return c.Key == ""
}

func (c ConfigMapKeySelector) String() string {
	return c.Name + "/" + c.Key
}

// +kubebuilder:object:generate=true
type SecretKeySelector struct {
	LocalObjectReference `json:",inline" yaml:",inline" protobuf:"bytes,1,opt,name=localObjectReference"`
	Key                  string `json:"key" yaml:"key" protobuf:"bytes,2,opt,name=key"`
}

func (s SecretKeySelector) IsEmpty() bool {
	return s.Key == ""
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
		return TextType
	case MysqlType:
		return TextType
	case PostgresType:
		return TextType
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

		if strings.HasPrefix(v, "helm://") {
			if len(strings.Split(v, "/")) != 4 {
				return fmt.Errorf("invalid helm reference: %s", v)
			}
			*e = EnvVar{
				ValueFrom: &EnvVarSource{
					HelmRef: &HelmRefKeySelector{
						LocalObjectReference: LocalObjectReference{
							Name: strings.Split(v, "/")[2],
						},
						Key: strings.Split(v, "/")[3],
					},
				}}
			return nil
		}

		if strings.HasPrefix(v, "serviceaccount://") {
			segments := strings.Split(v, "/")
			if len(segments) != 3 || segments[2] == "" {
				return fmt.Errorf("invalid service account reference: %s", v)
			}
			*e = EnvVar{
				ValueFrom: &EnvVarSource{
					ServiceAccount: &segments[2],
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

// +kubebuilder:object:generate=true
type ExtractionVar struct {
	Expr CelExpression `yaml:"expr,omitempty" json:"expr,omitempty"`

	// Value is a static value
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

func (t ExtractionVar) Empty() bool {
	return t.Value == "" && t.Expr == ""
}

func (t ExtractionVar) Eval(env map[string]any) (string, error) {
	if t.Value != "" {
		return t.Value, nil
	}

	return t.Expr.Eval(env)
}

// EnvVarResourceSelector is used to select a resource.
// At least one of the fields must be specified.
// +kubebuilder:object:generate=true
type EnvVarResourceSelector struct {
	Name ExtractionVar     `yaml:"name,omitempty" json:"name,omitempty"`
	Type ExtractionVar     `yaml:"type,omitempty" json:"type,omitempty"`
	Tags map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

func (t EnvVarResourceSelector) Empty() bool {
	return t.Name.Empty() && t.Type.Empty() && len(t.Tags) == 0
}

func (t EnvVarResourceSelector) Hydrate(env map[string]any) (*ResourceSelector, error) {
	var rs ResourceSelector

	if !t.Name.Empty() {
		name, err := t.Name.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate name: %v", err)
		}
		rs.Name = name
	}

	if !t.Type.Empty() {
		typ, err := t.Type.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate type: %v", err)
		}
		rs.Types = []string{typ}
	}

	if len(t.Tags) != 0 {
		rs.TagSelector = collections.SortedMap(t.Tags)
	}

	return &rs, nil
}
