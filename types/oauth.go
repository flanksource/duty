package types

import (
	"encoding/json"
	"fmt"

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

// PopulateFromProperties needs properties to be hydrated
func (o *OAuth) PopulateFromProperties(props map[string]string) error {
	o.ClientID.ValueStatic = props["clientID"]
	o.ClientSecret.ValueStatic = props["clientSecret"]
	o.TokenURL = props["tokenURL"]
	if props["scope"] != "" {
		if err := json.Unmarshal([]byte(props["scopes"]), &o.Scopes); err != nil {
			return fmt.Errorf("error unmarshaling scopes:%s in oauth: %w", props["scopes"], err)
		}
	}
	if props["params"] != "" {
		if err := json.Unmarshal([]byte(props["params"]), &o.Params); err != nil {
			return fmt.Errorf("error unmarshaling params:%s in oauth: %w", props["params"], err)
		}
	}
	return nil
}

func (o OAuth) AsProperties() JSONStringMap {
	var scopes, params string
	if o.Scopes != nil {
		scopes, _ = collections.StructToJSON(o.Scopes)
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
