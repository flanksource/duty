package connection

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

type PrometheusConnection struct {
	HTTPConnection `json:",inline" yaml:",inline"`
}

func (t *PrometheusConnection) FromModel(connection models.Connection) error {
	if connection.Type != models.ConnectionTypePrometheus {
		return fmt.Errorf("connection of type %s cannot be used with prometheus", connection.Type)
	}

	t.HTTPConnection.FromModel(connection)
	return nil
}

func (p *PrometheusConnection) Populate(ctx ConnectionContext) error {
	if _, err := p.HTTPConnection.Hydrate(ctx, ctx.GetNamespace()); err != nil {
		return err
	}

	return nil
}

func (p *PrometheusConnection) NewClient(ctx context.Context) (v1.API, error) {
	transport := &http.Transport{}
	if p.HTTPConnection.TLS.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	var roundTripper http.RoundTripper = transport
	if !p.HTTPConnection.Username.IsEmpty() || !p.HTTPConnection.Password.IsEmpty() {
		roundTripper = &basicAuthRoundTripper{
			username: p.HTTPConnection.Username.ValueStatic,
			password: p.HTTPConnection.Password.ValueStatic,
			base:     roundTripper,
		}
	}

	cfg := api.Config{
		Address:      p.HTTPConnection.URL,
		RoundTripper: roundTripper,
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return v1.NewAPI(client), nil
}

type basicAuthRoundTripper struct {
	username, password string
	base               http.RoundTripper
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(rt.username, rt.password)
	return rt.base.RoundTrip(req)
}
