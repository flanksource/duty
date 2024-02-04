package tracing

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/sdk/trace"
	// "go.opentelemetry.io/otel/trace"
)

func WithSkipSpan(ctx context.Context) context.Context {
	return context.WithValue(ctx, TracePaused, "true")
}

// CustomSampler struct holds different samplers.
type CustomSampler struct {
	defaultSampler trace.Sampler
	samplers       map[string]trace.Sampler
}

// NewCustomSampler creates a new instance of CustomSampler.
// `defaultSampler` is used when no specific sampler is found for a span name.
// `samplers` is a map where keys are span names and values are the specific samplers for those span names.
func NewCustomSampler(defaultSampler trace.Sampler, samplers map[string]trace.Sampler) *CustomSampler {
	return &CustomSampler{
		defaultSampler: defaultSampler,
		samplers:       samplers,
	}
}

// ShouldSample implements the Sampler interface.
// It delegates the decision to a sampler based on the span name.
func (cs *CustomSampler) ShouldSample(params trace.SamplingParameters) trace.SamplingResult {
	if skip := params.ParentContext.Value(TracePaused); skip != nil {
		return trace.SamplingResult{
			Decision:   trace.Drop,
			Attributes: nil,
		}
	}
	if sampler, ok := cs.samplers[params.Name]; ok {
		return sampler.ShouldSample(params)
	}
	return cs.defaultSampler.ShouldSample(params)
}

// Description returns a description of the sampler.
func (cs *CustomSampler) Description() string {
	return "CustomSampler"
}

// CounterSampler samples spans based on a counter, ensuring a percentage of spans are sampled.
type CounterSampler struct {
	percentage float64
	counter    atomic.Int64
	rate       int64
}

// NewCounterSampler creates a new CounterSampler.
// `percentage` should be a value between 0 and 100, representing the percentage of spans to sample.
func NewCounterSampler(percentage float64) *CounterSampler {
	return &CounterSampler{
		percentage: percentage,
		counter:    atomic.Int64{},
		rate:       int64(100.0 / percentage),
	}
}

// ShouldSample decides whether a span should be sampled based on a counter.
func (cs *CounterSampler) ShouldSample(params trace.SamplingParameters) trace.SamplingResult {
	// Atomically increment the counter
	count := cs.counter.Add(1)

	// Sample the span if the count modulo the rate equals 1 (to sample the first span in each cycle)
	if count%cs.rate == 1 {
		return trace.SamplingResult{
			Decision:   trace.RecordAndSample,
			Attributes: nil,
		}
	}

	return trace.SamplingResult{
		Decision:   trace.Drop,
		Attributes: nil,
	}
}

// Description returns the description of the sampler.
func (cs *CounterSampler) Description() string {
	return "CounterSampler"
}
