package types

import (
	"strings"

	"github.com/flanksource/commons/collections"
)

// +kubebuilder:object:generate=true
type OAuth struct {
	ClientID     EnvVar            `json:"clientID,omitempty"`
	ClientSecret EnvVar            `json:"clientSecret,omitempty"`
	Scopes       []string          `json:"scope,omitempty" yaml:"scope,omitempty"`
	TokenURL     string            `json:"tokenURL,omitempty" yaml:"tokenURL,omitempty"`
	Params       map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}

func (o OAuth) IsEmpty() bool {
	return o.ClientID.IsEmpty() || o.ClientSecret.IsEmpty() || o.TokenURL == ""
}

func (o OAuth) AsProperties() JSONStringMap {
	var scopes, params string
	if o.Scopes != nil {
		scopes = strings.Join(o.Scopes, ",")
	}
	if o.Params != nil {
		params, _ = collections.StructToJSON(o.Params)
	}
	return map[string]string{
		"clientID":     o.ClientID.String(),
		"clientSecret": o.ClientSecret.String(),
		"tokenURL":     o.TokenURL,
		"scopes":       scopes,
		"params":       params,
	}
}
