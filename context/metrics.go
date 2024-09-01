package context

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/text"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

var MetricsLogLevel = 5

func mapToSlice(c map[string]string) []any {
	args := []any{}
	for k, v := range c {
		if !lo.IsEmpty(v) {
			args = append(args, k, v)
		}
	}
	return args
}

type Histogram struct {
	Context   Context
	Name      string
	Histogram *prometheus.HistogramVec
	Labels    map[string]string
}

var ctxHistograms sync.Map

var LatencyBuckets = []float64{
	float64(10 * time.Millisecond),
	float64(100 * time.Millisecond),
	float64(500 * time.Millisecond),
	float64(1 * time.Second),
	float64(10 * time.Second),
}

var ShortLatencyBuckets = []float64{
	float64(10 * time.Millisecond),
	float64(100 * time.Millisecond),
	float64(500 * time.Millisecond),
}

var LongLatencyBuckets = []float64{
	float64(1 * time.Second),
	float64(10 * time.Second),
	float64(100 * time.Second),
	float64(1000 * time.Second),
}

func (k Context) Histogram(name string, buckets []float64, labels ...string) Histogram {
	labelMap := stringSliceToMap(labels)
	labelKeys := maps.Keys(labelMap)
	slices.Sort(labelKeys)
	key := strings.Join(append(labelKeys, name), ".")

	if histo, exists := ctxHistograms.Load(key); exists {
		return Histogram{
			Context:   k,
			Histogram: histo.(*prometheus.HistogramVec),
			Name:      name,
			Labels:    labelMap,
		}
	}

	histo := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Buckets: buckets,
	}, labelKeys)

	if err := prometheus.Register(histo); err != nil {
		k.Errorf("error registering histogram[%s/%v]: %v", name, labels, err)
	}

	ctxHistograms.Store(key, histo)
	return Histogram{
		Context:   k,
		Histogram: histo,
		Name:      name,
		Labels:    stringSliceToMap(labels),
	}
}

func (h *Histogram) Label(k, v string) Histogram {
	h.Labels[k] = v
	return *h
}

func (h Histogram) Record(duration time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			h.Context.Errorf("error observe to histogram[%s/%v]: %v", h.Name, h.Labels, r)
		}
	}()

	if duration > time.Millisecond*5 {
		if logger := logger.GetLogger("metrics." + h.Name); logger.IsLevelEnabled(4) {
			logger.WithValues(mapToSlice(h.Labels)...).V(MetricsLogLevel).Infof("%s", text.HumanizeDuration(duration))
		}
	}

	h.Histogram.With(prometheus.Labels(h.Labels)).Observe(float64(duration))
}

func (h Histogram) Since(s time.Time) {
	h.Record(time.Since(s))
}

type Counter struct {
	Context Context
	Name    string
	Labels  map[string]string
	Counter *prometheus.CounterVec
}

var ctxCounters sync.Map

func (k Context) Counter(name string, labels ...string) Counter {
	labelMap := stringSliceToMap(labels)
	labelKeys := maps.Keys(labelMap)
	slices.Sort(labelKeys)
	key := strings.Join(append(labelKeys, name), ".")

	if counter, exists := ctxCounters.Load(key); exists {
		return Counter{
			Context: k,
			Counter: counter.(*prometheus.CounterVec),
			Name:    name,
			Labels:  labelMap,
		}

	}

	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
	}, labelKeys)

	if err := prometheus.Register(counter); err != nil {
		k.Errorf("error registering counter[%s/%v]: %v", name, labels, err)
	}

	ctxCounters.Store(key, counter)
	return Counter{
		Context: k,
		Counter: counter,
		Name:    name,
		Labels:  labelMap,
	}
}

func (c Counter) Add(count int) {
	defer func() {
		if r := recover(); r != nil {
			c.Context.Errorf("error adding to counter[%s/%v]: %v", c.Name, c.Labels, r)
		}
	}()

	if logger := logger.GetLogger("metrics." + c.Name); logger.IsLevelEnabled(4) {
		logger.WithValues(mapToSlice(c.Labels)...).V(MetricsLogLevel).Infof("%d", count)
	}
	c.Counter.With(prometheus.Labels(c.Labels)).Add(float64(count))
}

func (c Counter) AddFloat(count float64) {
	defer func() {
		if r := recover(); r != nil {
			c.Context.Errorf("error adding to counter[%s/%v]: %v", c.Name, c.Labels, r)
		}
	}()

	if logger := logger.GetLogger("metrics." + c.Name); logger.IsLevelEnabled(4) {
		logger.WithValues(mapToSlice(c.Labels)...).V(MetricsLogLevel).Infof("%0.2f", count)
	}
	c.Counter.With(prometheus.Labels(c.Labels)).Add(count)
}

func (c *Counter) Label(k, v string) Counter {
	c.Labels[k] = v
	return *c
}

type Gauge struct {
	Context Context
	Name    string
	Labels  map[string]string
	Gauge   *prometheus.GaugeVec
}

var ctxGauges sync.Map

func (k Context) Gauge(name string, labels ...string) Gauge {
	labelMap := stringSliceToMap(labels)
	labelKeys := maps.Keys(labelMap)
	slices.Sort(labelKeys)
	key := strings.Join(append(labelKeys, name), ".")

	if gauge, exists := ctxGauges.Load(key); exists {
		return Gauge{
			Context: k,
			Gauge:   gauge.(*prometheus.GaugeVec),
			Name:    name,
			Labels:  labelMap,
		}
	}

	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
	}, labelKeys)

	if err := prometheus.Register(gauge); err != nil {
		k.Errorf("error registering gauge[%s/%v]: %v", name, labels, err)
	}

	ctxGauges.Store(key, gauge)
	return Gauge{
		Context: k,
		Gauge:   gauge,
		Name:    name,
		Labels:  labelMap,
	}
}

func (g Gauge) Set(count float64) {

	if logger := logger.GetLogger("metrics." + g.Name); logger.IsLevelEnabled(4) {
		logger.WithValues(mapToSlice(g.Labels)...).V(MetricsLogLevel).Infof("%0.2f", count)
	}

	g.Gauge.With(prometheus.Labels(g.Labels)).Set(count)
}

func (g Gauge) Add(count float64) {
	if logger := logger.GetLogger("metrics." + g.Name); logger.IsLevelEnabled(4) {
		logger.WithValues(mapToSlice(g.Labels)...).V(MetricsLogLevel).Infof("+%0.2f", count)
	}
	g.Gauge.With(prometheus.Labels(g.Labels)).Add(count)
}

func (g Gauge) Sub(count float64) {
	if logger := logger.GetLogger("metrics." + g.Name); logger.IsLevelEnabled(4) {
		logger.WithValues(mapToSlice(g.Labels)...).V(MetricsLogLevel).Infof("-%0.2f", count)
	}
	g.Gauge.With(prometheus.Labels(g.Labels)).Sub(count)
}
