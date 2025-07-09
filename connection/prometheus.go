package connection

import (
	"fmt"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// +kubebuilder:object:generate=true
type PrometheusConnection struct {
	HTTPConnection `json:",inline" yaml:",inline"`
}

func (t *PrometheusConnection) FromModel(connection models.Connection) error {
	if connection.Type != models.ConnectionTypePrometheus {
		return fmt.Errorf("connection of type %s cannot be used with prometheus, expected %s", connection.Type, models.ConnectionTypePrometheus)
	}

	if err := t.HTTPConnection.FromModel(connection); err != nil {
		return fmt.Errorf("failed to initialize HTTP connection: %w", err)
	}
	return nil
}

func (p *PrometheusConnection) Populate(ctx ConnectionContext) error {
	if _, err := p.HTTPConnection.Hydrate(ctx, ctx.GetNamespace()); err != nil {
		return err
	}

	return nil
}

func (p *PrometheusConnection) NewClient(ctx context.Context) (v1.API, error) {
	cfg := api.Config{
		Address:      p.HTTPConnection.URL,
		RoundTripper: p.HTTPConnection.Transport(),
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return v1.NewAPI(client), nil
}
