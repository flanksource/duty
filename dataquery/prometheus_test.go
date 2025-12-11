package dataquery

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
)

func TestToPrometheusRange(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		g := NewWithT(t)
		now := time.Date(2024, time.April, 10, 12, 0, 0, 0, time.UTC)

		pr := PrometheusRange{
			Start: "now-1h",
			End:   "now",
			Step:  "30s",
		}

		got, err := pr.toPrometheusRange(now)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(got).To(Equal(promv1.Range{
			Start: now.Add(-1 * time.Hour),
			End:   now,
			Step:  30 * time.Second,
		}))
	})

	t.Run("missing fields", func(t *testing.T) {
		g := NewWithT(t)
		now := time.Now()

		_, err := (PrometheusRange{}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("start time is required")))

		_, err = (PrometheusRange{Start: "now"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("end time is required")))

		_, err = (PrometheusRange{Start: "now", End: "now"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("step is required")))
	})

	t.Run("invalid values", func(t *testing.T) {
		g := NewWithT(t)
		now := time.Now()

		_, err := (PrometheusRange{Start: "bad", End: "now", Step: "30s"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("invalid prometheus range start time")))

		_, err = (PrometheusRange{Start: "now-1h", End: "bad", Step: "30s"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("invalid prometheus range end time")))

		_, err = (PrometheusRange{Start: "now-1h", End: "now", Step: "not-a-duration"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("invalid prometheus range step")))

		_, err = (PrometheusRange{Start: "now-1h", End: "now", Step: "0s"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("must be greater than zero")))

		_, err = (PrometheusRange{Start: "now", End: "now-1m", Step: "30s"}).toPrometheusRange(now)
		g.Expect(err).To(MatchError(ContainSubstring("end time must be after start time")))
	})
}

func TestRowFromMetric(t *testing.T) {
	t.Run("all labels returned by default", func(t *testing.T) {
		g := NewWithT(t)
		metric := model.Metric{
			"__name__": "up",
			"instance": "localhost:9090",
			"job":      "prometheus",
		}

		row := rowFromMetric(metric, nil)
		g.Expect(row).To(HaveLen(len(metric)))
		g.Expect(row).To(HaveKeyWithValue("__name__", "up"))
		g.Expect(row).To(HaveKeyWithValue("instance", "localhost:9090"))
		g.Expect(row).To(HaveKeyWithValue("job", "prometheus"))
	})

	t.Run("filters to match labels", func(t *testing.T) {
		g := NewWithT(t)
		metric := model.Metric{
			"__name__": "up",
			"instance": "localhost:9090",
			"job":      "prometheus",
		}

		row := rowFromMetric(metric, []string{"job"})
		g.Expect(row).To(HaveLen(1))
		g.Expect(row).To(HaveKeyWithValue("job", "prometheus"))
	})

	t.Run("ignores non-existent labels", func(t *testing.T) {
		g := NewWithT(t)
		metric := model.Metric{
			"instance": "localhost:9090",
		}

		row := rowFromMetric(metric, []string{"job"})
		g.Expect(row).To(BeEmpty())
	})
}

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
