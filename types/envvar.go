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
	ValueStatic string        `json:"value,omitempty" yaml:"value,omitempty" protobuf:"bytes,2,opt,name=value" template:"true"`
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
				},
			}
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
				},
			}
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
				},
			}
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
				},
			}
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
type ValueExpression struct {
	Expr CelExpression `yaml:"expr,omitempty" json:"expr,omitempty"`

	// Value is a static value
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

func (t ValueExpression) Empty() bool {
	return t.Value == "" && t.Expr == ""
}

func (t ValueExpression) Eval(env map[string]any) (string, error) {
	if t.Value != "" {
		return t.Value, nil
	}

	return t.Expr.Eval(env)
}

// EnvVarResourceSelector is used to select a resource.
// At least one of the fields must be specified.
// +kubebuilder:object:generate=true
type EnvVarResourceSelector struct {
	Agent         ValueExpression   `yaml:"agent,omitempty" json:"agent,omitempty"`
	Scope         string            `yaml:"scope,omitempty" json:"scope,omitempty"`
	Cache         string            `yaml:"cache,omitempty" json:"cache,omitempty"`
	ID            ValueExpression   `yaml:"id,omitempty" json:"id,omitempty"`
	Name          ValueExpression   `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace     ValueExpression   `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Types         []ValueExpression `yaml:"types,omitempty" json:"types,omitempty"`
	Statuses      []ValueExpression `yaml:"statuses,omitempty" json:"statuses,omitempty"`
	Healths       []ValueExpression `yaml:"healths,omitempty" json:"healths,omitempty"`
	TagSelector   ValueExpression   `yaml:"tagSelector,omitempty" json:"tagSelector,omitempty"`
	LabelSelector ValueExpression   `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	FieldSelector ValueExpression   `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`
}

func (t EnvVarResourceSelector) Empty() bool {
	return t.Agent.Empty() && t.Scope == "" && t.Cache == "" && t.ID.Empty() &&
		t.Name.Empty() && t.Namespace.Empty() && len(t.Types) == 0 && len(t.Statuses) == 0 &&
		len(t.Healths) == 0 && t.TagSelector.Empty() && t.LabelSelector.Empty() && t.FieldSelector.Empty()
}

func (t EnvVarResourceSelector) Hydrate(env map[string]any) (*ResourceSelector, error) {
	rs := ResourceSelector{
		Scope: t.Scope,
		Cache: t.Cache,
	}

	if !t.Agent.Empty() {
		agent, err := t.Agent.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate agent: %v", err)
		}
		rs.Agent = agent
	}

	if !t.ID.Empty() {
		id, err := t.ID.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate id: %v", err)
		}
		rs.ID = id
	}

	if !t.Name.Empty() {
		name, err := t.Name.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate name: %v", err)
		}
		rs.Name = name
	}

	if !t.Namespace.Empty() {
		namespace, err := t.Namespace.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate namespace: %v", err)
		}
		rs.Namespace = namespace
	}

	if len(t.Types) > 0 {
		rs.Types = make([]string, len(t.Types))
		for i, typeExpr := range t.Types {
			if !typeExpr.Empty() {
				typeStr, err := typeExpr.Eval(env)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate type at index %d: %v", i, err)
				}
				rs.Types[i] = typeStr
			}
		}
	}

	if len(t.Statuses) > 0 {
		rs.Statuses = make([]string, len(t.Statuses))
		for i, statusExpr := range t.Statuses {
			if !statusExpr.Empty() {
				statusStr, err := statusExpr.Eval(env)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate status at index %d: %v", i, err)
				}
				rs.Statuses[i] = statusStr
			}
		}
	}

	if len(t.Healths) > 0 {
		for i, expr := range t.Healths {
			if !expr.Empty() {
				result, err := expr.Eval(env)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate health at index %d: %v", i, err)
				}
				rs.Health.Add(result)
			}
		}
	}

	if !t.TagSelector.Empty() {
		tagSelector, err := t.TagSelector.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate tagSelector: %v", err)
		}
		rs.TagSelector = tagSelector
	}

	if !t.LabelSelector.Empty() {
		labelSelector, err := t.LabelSelector.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate labelSelector: %v", err)
		}
		rs.LabelSelector = labelSelector
	}

	if !t.FieldSelector.Empty() {
		fieldSelector, err := t.FieldSelector.Eval(env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate fieldSelector: %v", err)
		}
		rs.FieldSelector = fieldSelector
	}

	return &rs, nil
}
