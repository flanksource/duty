package types

import (
	"strings"
)

// +kubebuilder:object:generate=true
type Authentication struct {
	Username EnvVar `yaml:"username,omitempty" json:"username,omitempty"`
	Password EnvVar `yaml:"password,omitempty" json:"password,omitempty"`
}

func (auth Authentication) IsEmpty() bool {
	return (auth.Username.IsEmpty() && auth.Password.IsEmpty())
}

func (auth Authentication) GetUsername() string {
	return auth.Username.ValueStatic
}

func (auth Authentication) GetPassword() string {
	return auth.Password.ValueStatic
}

func (auth Authentication) GetDomain() string {
	parts := strings.Split(auth.GetUsername(), "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
