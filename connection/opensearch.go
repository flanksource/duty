package connection

import (
	"strconv"
	"strings"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
)

// +kubebuilder:object:generate=true
type OpensearchConnection struct {
	ConnectionName      string `json:"connection,omitempty" yaml:"connection,omitempty"`
	types.HTTPBasicAuth `json:",inline"`
	URLs                []string `json:"urls,omitempty" yaml:"urls,omitempty"`
	Index               string   `json:"index,omitempty" yaml:"index,omitempty"`
	InsecureSkipVerify  bool     `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
}

func (c OpensearchConnection) ToModel() models.Connection {
	return models.Connection{
		Type:     models.ConnectionTypeOpenSearch,
		URL:      lo.FirstOrEmpty(c.URLs),
		Username: c.Username.String(),
		Password: c.Password.String(),
		Properties: map[string]string{
			"urls":         strings.Join(c.URLs, ","),
			"index":        c.Index,
			"insecure_tls": strconv.FormatBool(c.InsecureSkipVerify),
		},
	}
}

func NewOpenSearchClient() {}
