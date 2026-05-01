package connection

import (
	"encoding/json"
	"fmt"
	netHTTP "net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/commons/har"
	commonsHTTP "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/logger/httpretty"
	"github.com/flanksource/commons/properties"
	"github.com/patrickmn/go-cache"
)

// tokenSafetyMargin is the duration to deduct when caching tokens.
// Example: if a token expires in 15 minutes then we cache it with a
// TTL of 15 minutes - tokenSafetyMargin = 10 minutes.
//
// This is to prevent using a token that immediately expires.
const tokenSafetyMargin = time.Minute * 5

// caches cloud tokens. eg: EKS token, GKE token, ...
var tokenCache = cache.New(time.Hour, time.Hour)

func tokenCacheKey(cloud string, cred any, identifiers string) string {
	switch v := cred.(type) {
	case string:
		return fmt.Sprintf("%s-%s-%s", cloud, v, identifiers)
	default:
		m, _ := json.Marshal(v)
		return fmt.Sprintf("%s-%s-%s", cloud, m, identifiers)
	}
}

type observabilityContext interface {
	HARCollector() *har.Collector
	EffectiveHARCollector(feature string, explicit *har.Collector) *har.Collector
	EffectiveHARLevel(feature string) (logger.LogLevel, string)
	HTTPLoggingContent(feature string) (bool, bool)
}

func effectiveHARCollector(ctx any, feature string, explicit *har.Collector) *har.Collector {
	if c, ok := ctx.(observabilityContext); ok {
		return c.EffectiveHARCollector(feature, explicit)
	}
	return explicit
}

func applyHTTPObservability(ctx any, feature string, base netHTTP.RoundTripper, explicit *har.Collector) netHTTP.RoundTripper {
	if base == nil {
		base = netHTTP.DefaultTransport
	}
	if middleware := harCollectorMiddleware(ctx, feature, explicit); middleware != nil {
		base = middleware(base)
	}
	if c, ok := ctx.(observabilityContext); ok {
		headers, bodies := c.HTTPLoggingContent(feature)
		base = httpLoggerWithContent(base, headers, bodies)
	}
	return base
}

func httpObservabilityMiddleware(ctx any, feature string, explicit *har.Collector) middlewares.Middleware {
	if effectiveHARCollector(ctx, feature, explicit) != nil {
		return func(rt netHTTP.RoundTripper) netHTTP.RoundTripper {
			return applyHTTPObservability(ctx, feature, rt, explicit)
		}
	}
	if c, ok := ctx.(observabilityContext); ok {
		headers, _ := c.HTTPLoggingContent(feature)
		if headers {
			return func(rt netHTTP.RoundTripper) netHTTP.RoundTripper {
				return applyHTTPObservability(ctx, feature, rt, explicit)
			}
		}
	}
	return nil
}

func applyHTTPClientObservability(ctx any, feature string, client *commonsHTTP.Client, explicit *har.Collector) middlewares.Middleware {
	if client == nil {
		return nil
	}

	var tokenTransport middlewares.Middleware
	level := logger.Info
	if c, ok := ctx.(observabilityContext); ok {
		level, _ = c.EffectiveHARLevel(feature)
	}
	if explicit != nil && level < logger.Debug {
		level = logger.Trace
	}

	if collector := effectiveHARCollector(ctx, feature, explicit); collector != nil && level >= logger.Debug {
		if level >= logger.Trace {
			client.HARCollector(collector)
		} else {
			middleware := metadataHARMiddleware(collector)
			client.Use(middleware)
			tokenTransport = middleware
		}
	}

	if c, ok := ctx.(observabilityContext); ok {
		headers, bodies := c.HTTPLoggingContent(feature)
		if headers {
			logMiddleware := func(rt netHTTP.RoundTripper) netHTTP.RoundTripper {
				return httpLoggerWithContent(rt, headers, bodies)
			}
			client.Use(logMiddleware)
			if tokenTransport == nil {
				tokenTransport = logMiddleware
			} else {
				existing := tokenTransport
				tokenTransport = func(rt netHTTP.RoundTripper) netHTTP.RoundTripper {
					return logMiddleware(existing(rt))
				}
			}
		}
	}

	return tokenTransport
}

func harTokenTransport(ctx any, feature string, explicit *har.Collector) middlewares.Middleware {
	return func(rt netHTTP.RoundTripper) netHTTP.RoundTripper {
		return applyHTTPObservability(ctx, feature, rt, explicit)
	}
}

func harCollectorMiddleware(ctx any, feature string, explicit *har.Collector) middlewares.Middleware {
	level := logger.Info
	if c, ok := ctx.(observabilityContext); ok {
		level, _ = c.EffectiveHARLevel(feature)
	}
	if explicit != nil && level < logger.Debug {
		level = logger.Trace
	}

	collector := effectiveHARCollector(ctx, feature, explicit)
	if collector == nil || level < logger.Debug {
		return nil
	}
	if level >= logger.Trace {
		return collector.Middleware()
	}
	return metadataHARMiddleware(collector)
}

func httpLoggerWithContent(rt netHTTP.RoundTripper, headers, bodies bool) netHTTP.RoundTripper {
	if !headers {
		return rt
	}

	l := &httpretty.Logger{
		Time:            true,
		TLS:             true,
		Auth:            true,
		RequestHeader:   true,
		RequestBody:     bodies,
		ResponseHeader:  true,
		ResponseBody:    bodies,
		Colors:          true,
		Formatters:      []httpretty.Formatter{&httpretty.JSONFormatter{}},
		MaxResponseBody: int64(properties.Int(4*1024, "http.log.response.body.length")),
	}
	l.SkipHeader(logger.SensitiveHeaders)
	return l.RoundTripper(rt)
}

func metadataHARMiddleware(collector *har.Collector) middlewares.Middleware {
	return func(next netHTTP.RoundTripper) netHTTP.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *netHTTP.Request) (*netHTTP.Response, error) {
			started := time.Now()
			entry := &har.Entry{
				StartedDateTime: started.UTC().Format(time.RFC3339),
				Request: har.Request{
					Method:      req.Method,
					URL:         req.URL.String(),
					HTTPVersion: harHTTPVersion(req.Proto),
					Cookies:     []har.Cookie{},
					Headers:     toHARHeaders(logger.SanitizeHeaders(req.Header)),
					QueryString: toHARQueryString(req.URL.Query()),
					HeadersSize: -1,
					BodySize:    -1,
				},
			}

			waitStart := time.Now()
			resp, err := next.RoundTrip(req)
			waitMs := float64(time.Since(waitStart).Microseconds()) / 1000.0

			entry.Timings = har.Timings{Wait: waitMs}
			entry.Time = waitMs
			if resp != nil {
				entry.Response = har.Response{
					Status:      resp.StatusCode,
					StatusText:  resp.Status,
					HTTPVersion: harHTTPVersion(resp.Proto),
					Cookies:     []har.Cookie{},
					Headers:     toHARHeaders(logger.SanitizeHeaders(resp.Header)),
					Content:     har.Content{Size: -1},
					RedirectURL: "",
					HeadersSize: -1,
					BodySize:    -1,
				}
			} else {
				// Transport error: no response object. Use -1 sentinels (HAR spec
				// for "size unknown") so consumers don't read Status=0 as a
				// successful empty response.
				entry.Response = har.Response{
					Cookies:     []har.Cookie{},
					Headers:     []har.Header{},
					Content:     har.Content{Size: -1},
					HeadersSize: -1,
					BodySize:    -1,
				}
			}

			collector.Add(entry)
			return resp, err
		})
	}
}

func toHARHeaders(h netHTTP.Header) []har.Header {
	headers := make([]har.Header, 0, len(h))
	for name, vals := range h {
		for _, v := range vals {
			headers = append(headers, har.Header{Name: name, Value: v})
		}
	}
	return headers
}

func toHARQueryString(q url.Values) []har.QueryString {
	qs := make([]har.QueryString, 0, len(q))
	for k, vs := range q {
		for _, v := range vs {
			qs = append(qs, har.QueryString{Name: k, Value: v})
		}
	}
	return qs
}

func harHTTPVersion(proto string) string {
	if strings.TrimSpace(proto) == "" {
		return "HTTP/1.1"
	}
	return proto
}
