package dataquery

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

func TestTransformPrometheusResult_NilResult(t *testing.T) {
	g := NewWithT(t)

	result, err := transformPrometheusResult(nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_VectorSamples(t *testing.T) {
	g := NewWithT(t)

	vector := model.Vector{
		&model.Sample{
			Metric: model.Metric{
				"__name__": "test_metric",
				"instance": "localhost:8080",
				"job":      "test_job",
			},
			Value: 42.5,
		},
		&model.Sample{
			Metric: model.Metric{
				"__name__": "another_metric",
				"instance": "localhost:9090",
			},
			Value: 100.0,
		},
	}

	result, err := transformPrometheusResult(vector)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(2))

	// Check first sample
	g.Expect(result[0]["__name__"]).To(Equal("test_metric"))
	g.Expect(result[0]["instance"]).To(Equal("localhost:8080"))
	g.Expect(result[0]["job"]).To(Equal("test_job"))
	g.Expect(result[0]["value"]).To(Equal(42.5))

	// Check second sample
	g.Expect(result[1]["__name__"]).To(Equal("another_metric"))
	g.Expect(result[1]["instance"]).To(Equal("localhost:9090"))
	g.Expect(result[1]["value"]).To(Equal(100.0))
}

func TestTransformPrometheusResult_EmptyVector(t *testing.T) {
	g := NewWithT(t)

	vector := model.Vector{}
	result, err := transformPrometheusResult(vector)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_MatrixSamples(t *testing.T) {
	g := NewWithT(t)

	now := time.Now()
	matrix := model.Matrix{
		&model.SampleStream{
			Metric: model.Metric{
				"__name__": "test_metric",
				"instance": "localhost:8080",
			},
			Values: []model.SamplePair{
				{
					Timestamp: model.Time(now.Unix() * 1000),
					Value:     10.5,
				},
				{
					Timestamp: model.Time((now.Unix() + 60) * 1000),
					Value:     20.0,
				},
			},
		},
	}

	result, err := transformPrometheusResult(matrix)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(2))

	// Check first data point
	g.Expect(result[0]["__name__"]).To(Equal("test_metric"))
	g.Expect(result[0]["instance"]).To(Equal("localhost:8080"))
	g.Expect(result[0]["timestamp"]).To(Equal(int64(now.Unix())))
	g.Expect(result[0]["value"]).To(Equal(10.5))

	// Check second data point
	g.Expect(result[1]["__name__"]).To(Equal("test_metric"))
	g.Expect(result[1]["instance"]).To(Equal("localhost:8080"))
	g.Expect(result[1]["timestamp"]).To(Equal(int64(now.Unix() + 60)))
	g.Expect(result[1]["value"]).To(Equal(20.0))
}

func TestTransformPrometheusResult_EmptyMatrix(t *testing.T) {
	g := NewWithT(t)

	matrix := model.Matrix{}
	result, err := transformPrometheusResult(matrix)
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

	result, err := transformPrometheusResult(matrix)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestTransformPrometheusResult_ScalarValue(t *testing.T) {
	g := NewWithT(t)

	scalar := &model.Scalar{
		Value: 123.456,
	}

	result, err := transformPrometheusResult(scalar)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(123.456))
}

func TestTransformPrometheusResult_ZeroScalarValue(t *testing.T) {
	g := NewWithT(t)

	scalar := &model.Scalar{
		Value: 0.0,
	}

	result, err := transformPrometheusResult(scalar)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(0.0))
}

func TestTransformPrometheusResult_NegativeScalarValue(t *testing.T) {
	g := NewWithT(t)

	scalar := &model.Scalar{
		Value: -42.5,
	}

	result, err := transformPrometheusResult(scalar)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(-42.5))
}

func TestTransformPrometheusResult_StringValue(t *testing.T) {
	g := NewWithT(t)

	str := &model.String{
		Value: "test_string_value",
	}

	result, err := transformPrometheusResult(str)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal("test_string_value"))
}

func TestTransformPrometheusResult_EmptyStringValue(t *testing.T) {
	g := NewWithT(t)

	str := &model.String{
		Value: "",
	}

	result, err := transformPrometheusResult(str)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal(""))
}

func TestTransformPrometheusResult_StringWithSpecialChars(t *testing.T) {
	g := NewWithT(t)

	str := &model.String{
		Value: "test with spaces and symbols !@#$%",
	}

	result, err := transformPrometheusResult(str)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]["value"]).To(Equal("test with spaces and symbols !@#$%"))
}
