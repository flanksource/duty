package dataquery

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
)

func TestTransformPrometheusResult_NilResult(t *testing.T) {
	g := NewWithT(t)

	result, err := transformPrometheusResult(nil, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_EmptyVector(t *testing.T) {
	g := NewWithT(t)

	vector := model.Vector{}
	result, err := transformPrometheusResult(vector, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_EmptyMatrix(t *testing.T) {
	g := NewWithT(t)

	matrix := model.Matrix{}
	result, err := transformPrometheusResult(matrix, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_MatrixWithEmptyStream(t *testing.T) {
	g := NewWithT(t)

	matrix := model.Matrix{
		&model.SampleStream{
			Metric: model.Metric{
				"__name__": "test_metric",
			},
			Values: []model.SamplePair{},
		},
	}

	result, err := transformPrometheusResult(matrix, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_ZeroScalarValue(t *testing.T) {
	g := NewWithT(t)

	scalar := &model.Scalar{
		Value: 0.0,
	}

	result, err := transformPrometheusResult(scalar, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(0.0))
}

func TestTransformPrometheusResult_NegativeScalarValue(t *testing.T) {
	g := NewWithT(t)

	scalar := &model.Scalar{
		Value: -42.5,
	}

	result, err := transformPrometheusResult(scalar, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(-42.5))
}

func TestTransformPrometheusResult_EmptyStringValue(t *testing.T) {
	g := NewWithT(t)

	str := &model.String{
		Value: "",
	}

	result, err := transformPrometheusResult(str, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(""))
}

func TestTransformPrometheusResult_StringWithSpecialChars(t *testing.T) {
	g := NewWithT(t)

	str := &model.String{
		Value: "test with spaces and symbols !@#$%",
	}

	result, err := transformPrometheusResult(str, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal("test with spaces and symbols !@#$%"))
}

func TestTransformPrometheusResult_VectorSamples(t *testing.T) {
	t.Skip("Demo instance is not available")
	g := NewWithT(t)

	ctx := context.New()

	// Create PrometheusQuery with demo endpoint
	pq := PrometheusQuery{
		PrometheusConnection: connection.PrometheusConnection{
			HTTPConnection: connection.HTTPConnection{
				URL: "https://prometheus.demo.prometheus.io",
			},
		},
		Query: "up",
	}

	result, err := executePrometheusQuery(ctx, pq)
	g.Expect(err).ToNot(HaveOccurred())

	want := []QueryResultRow{
		{"__name__": "up", "instance": "http://localhost:9100", "job": "blackbox", "value": float64(1)},
		{"__name__": "up", "instance": "localhost:2019", "job": "caddy", "value": float64(1)},
		{"__name__": "up", "instance": "demo.prometheus.io:8998", "job": "random", "value": float64(1)},
		{"__name__": "up", "env": "demo", "instance": "demo.prometheus.io:9100", "job": "node", "value": float64(1)},
		{"__name__": "up", "instance": "demo.prometheus.io:9090", "job": "prometheus", "value": float64(1)},
		{"__name__": "up", "env": "demo", "instance": "demo.prometheus.io:8080", "job": "cadvisor", "value": float64(1)},
		{"__name__": "up", "env": "demo", "instance": "demo.prometheus.io:9093", "job": "alertmanager", "value": float64(1)},
		{"__name__": "up", "instance": "demo.prometheus.io:8997", "job": "random", "value": float64(1)},
		{"__name__": "up", "instance": "demo.prometheus.io:8996", "job": "random", "value": float64(1)},
		{"__name__": "up", "instance": "demo.prometheus.io:3000", "job": "grafana", "value": float64(1)},
		{"__name__": "up", "instance": "demo.prometheus.io:8999", "job": "random", "value": float64(1)},
	}
	g.Expect(result).To(ConsistOf(want))
}

func TestTransformPrometheusResult_MatrixSamples(t *testing.T) {
	t.Skip("Prometheus demo endpoint is not available")
	g := NewWithT(t)

	ctx := context.New()

	// Create PrometheusQuery with demo endpoint for range query
	pq := PrometheusQuery{
		PrometheusConnection: connection.PrometheusConnection{
			HTTPConnection: connection.HTTPConnection{
				URL: "https://prometheus.demo.prometheus.io",
			},
		},
		Query: `up{instance="demo.prometheus.io:3000"}[5m]`,
	}

	result, err := executePrometheusQuery(ctx, pq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeEmpty())
	g.Expect(result).To(HaveLen(20))

	// Verify that we have real prometheus data with timestamps
	for _, row := range result {
		g.Expect(row).To(HaveKey("__name__"))
		g.Expect(row).To(HaveKey("value"))
		g.Expect(row).To(HaveKey("timestamp"))
		g.Expect(row).To(HaveKey("instance"))
		g.Expect(row["__name__"]).To(Equal("up"))
		g.Expect(row["timestamp"]).To(BeAssignableToTypeOf(time.Time{}))
	}
}

func TestTransformPrometheusResult_ScalarValue(t *testing.T) {
	t.Skip("Prometheus demo endpoint is not available")
	g := NewWithT(t)

	ctx := context.New()

	// Create PrometheusQuery with demo endpoint for scalar query
	pq := PrometheusQuery{
		PrometheusConnection: connection.PrometheusConnection{
			HTTPConnection: connection.HTTPConnection{
				URL: "https://prometheus.demo.prometheus.io",
			},
		},
		Query: "scalar(count(up))",
	}

	result, err := executePrometheusQuery(ctx, pq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(float64(11)))
}
