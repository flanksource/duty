package types

import "github.com/flanksource/commons/har"

type ClientOptions struct {
	HARCollector *har.Collector
	Feature      string
}

type ClientOption func(*ClientOptions)

func WithHARCollector(c *har.Collector) ClientOption {
	return func(o *ClientOptions) { o.HARCollector = c }
}

func WithFeature(name string) ClientOption {
	return func(o *ClientOptions) { o.Feature = name }
}

func NewClientOptions(opts ...ClientOption) ClientOptions {
	var o ClientOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
