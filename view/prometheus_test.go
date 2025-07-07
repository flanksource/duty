package view

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var _ = Describe("transformPrometheusResult", func() {
	Context("when result is nil", func() {
		It("should return empty slice", func() {
			result, err := transformPrometheusResult(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Context("when result is model.Vector", func() {
		It("should transform vector samples correctly", func() {
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
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			// Check first sample
			Expect(result[0]["__name__"]).To(Equal("test_metric"))
			Expect(result[0]["instance"]).To(Equal("localhost:8080"))
			Expect(result[0]["job"]).To(Equal("test_job"))
			Expect(result[0]["value"]).To(Equal(42.5))

			// Check second sample
			Expect(result[1]["__name__"]).To(Equal("another_metric"))
			Expect(result[1]["instance"]).To(Equal("localhost:9090"))
			Expect(result[1]["value"]).To(Equal(100.0))
		})

		It("should handle empty vector", func() {
			vector := model.Vector{}
			result, err := transformPrometheusResult(vector)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Context("when result is model.Matrix", func() {
		It("should transform matrix samples correctly", func() {
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
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			// Check first data point
			Expect(result[0]["__name__"]).To(Equal("test_metric"))
			Expect(result[0]["instance"]).To(Equal("localhost:8080"))
			Expect(result[0]["timestamp"]).To(Equal(int64(now.Unix())))
			Expect(result[0]["value"]).To(Equal(10.5))

			// Check second data point
			Expect(result[1]["__name__"]).To(Equal("test_metric"))
			Expect(result[1]["instance"]).To(Equal("localhost:8080"))
			Expect(result[1]["timestamp"]).To(Equal(int64(now.Unix() + 60)))
			Expect(result[1]["value"]).To(Equal(20.0))
		})

		It("should handle empty matrix", func() {
			matrix := model.Matrix{}
			result, err := transformPrometheusResult(matrix)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should handle matrix with empty sample stream", func() {
			matrix := model.Matrix{
				&model.SampleStream{
					Metric: model.Metric{
						"__name__": "test_metric",
					},
					Values: []model.SamplePair{},
				},
			}

			result, err := transformPrometheusResult(matrix)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Context("when result is *model.Scalar", func() {
		It("should transform scalar value correctly", func() {
			scalar := &model.Scalar{
				Value: 123.456,
			}

			result, err := transformPrometheusResult(scalar)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0]["value"]).To(Equal(123.456))
		})

		It("should handle zero scalar value", func() {
			scalar := &model.Scalar{
				Value: 0.0,
			}

			result, err := transformPrometheusResult(scalar)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0]["value"]).To(Equal(0.0))
		})

		It("should handle negative scalar value", func() {
			scalar := &model.Scalar{
				Value: -42.5,
			}

			result, err := transformPrometheusResult(scalar)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0]["value"]).To(Equal(-42.5))
		})
	})

	Context("when result is *model.String", func() {
		It("should transform string value correctly", func() {
			str := &model.String{
				Value: "test_string_value",
			}

			result, err := transformPrometheusResult(str)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0]["value"]).To(Equal("test_string_value"))
		})

		It("should handle empty string value", func() {
			str := &model.String{
				Value: "",
			}

			result, err := transformPrometheusResult(str)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0]["value"]).To(Equal(""))
		})

		It("should handle string with special characters", func() {
			str := &model.String{
				Value: "test with spaces and symbols !@#$%",
			}

			result, err := transformPrometheusResult(str)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0]["value"]).To(Equal("test with spaces and symbols !@#$%"))
		})
	})
})
