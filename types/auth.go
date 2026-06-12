package types

import (
	"strings"
)

// +kubebuilder:object:generate=true
type HTTPBasicAuth struct {
	Authentication `yaml:",inline" json:",inline" template:"true"`
	NTLM           bool `yaml:"ntlm,omitempty" json:"ntlm,omitempty"`
	NTLMV2         bool `yaml:"ntlmv2,omitempty" json:"ntlmv2,omitempty"`
	Digest         bool `yaml:"digest,omitempty" json:"digest,omitempty"`
}

// +kubebuilder:object:generate=true
type Authentication struct {
	Username EnvVar `yaml:"username,omitempty" json:"username,omitempty" template:"true"`
	Password EnvVar `yaml:"password,omitempty" json:"password,omitempty" template:"true"`
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
