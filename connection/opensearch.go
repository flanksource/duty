package connection

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	opensearch "github.com/opensearch-project/opensearch-go/v2"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type OpensearchConnection struct {
	ConnectionName      string `json:"connection,omitempty" yaml:"connection,omitempty"`
	types.HTTPBasicAuth `json:",inline"`
	// +kubebuilder:validation:MinItems=1
	URLs               []string `json:"urls,omitempty" yaml:"urls,omitempty"`
	Index              string   `json:"index,omitempty" yaml:"index,omitempty"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
}

func (c *OpensearchConnection) FromModel(connection models.Connection) error {
	c.ConnectionName = connection.Name

	if err := c.Username.Scan(connection.Username); err != nil {
		return fmt.Errorf("error scanning username: %w", err)
	}
	if err := c.Password.Scan(connection.Password); err != nil {
		return fmt.Errorf("error scanning password: %w", err)
	}

	// Parse URLs from properties or use the main URL
	if urlsStr := connection.Properties["urls"]; urlsStr != "" {
		c.URLs = strings.Split(urlsStr, ",")
	} else if connection.URL != "" {
		c.URLs = []string{connection.URL}
	} else {
		return fmt.Errorf("no urls found")
	}

	c.Index = connection.Properties["index"]

	if insecureTLS := connection.Properties["insecure_tls"]; insecureTLS != "" {
		c.InsecureSkipVerify, _ = strconv.ParseBool(insecureTLS)
	} else {
		c.InsecureSkipVerify = connection.InsecureTLS
	}

	return nil
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

func (c *OpensearchConnection) Hydrate(ctx ConnectionContext) error {
	if c.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(c.ConnectionName)
		if err != nil {
			return fmt.Errorf("could not hydrate connection[%s]: %w", c.ConnectionName, err)
		}
		if connection == nil {
			return fmt.Errorf("connection[%s] not found", c.ConnectionName)
		}
		if err := c.FromModel(*connection); err != nil {
			return err
		}
	}

	ns := ctx.GetNamespace()

	if !c.Username.IsEmpty() {
		if v, err := ctx.GetEnvValueFromCache(c.Username, ns); err != nil {
			return fmt.Errorf("could not get opensearch username from env var: %w", err)
		} else {
			c.Username.ValueStatic = v
		}
	}

	if !c.Password.IsEmpty() {
		if v, err := ctx.GetEnvValueFromCache(c.Password, ns); err != nil {
			return fmt.Errorf("could not get opensearch password from env var: %w", err)
		} else {
			c.Password.ValueStatic = v
		}
	}

	return nil
}

// Client creates and returns an OpenSearch client.
// NOTE: Must be run on a hydrated OpensearchConnection.
func (c *OpensearchConnection) Client() (*opensearch.Client, error) {
	if len(c.URLs) == 0 {
		return nil, fmt.Errorf("opensearch connection urls cannot be empty")
	}

	cfg := opensearch.Config{
		Addresses: c.URLs,
	}

	if !c.HTTPBasicAuth.IsEmpty() {
		cfg.Username = c.GetUsername()
		cfg.Password = c.GetPassword()
	}

	if c.InsecureSkipVerify {
		cfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	client, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating opensearch client: %w", err)
	}

	return client, nil
}

func NewOpenSearchClient(ctx context.Context, connection models.Connection) (*opensearch.Client, error) {
	var conn OpensearchConnection
	if err := conn.FromModel(connection); err != nil {
		return nil, fmt.Errorf("error creating opensearch connection from model: %w", err)
	}

	if err := conn.Hydrate(ctx); err != nil {
		return nil, fmt.Errorf("error hydrating opensearch connection: %w", err)
	}

	return conn.Client()
}
