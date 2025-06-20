package logs

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/tokenizer"
	"github.com/timberio/go-datemath"
)

type LogLine struct {
	ID            string            `json:"id,omitempty"`
	FirstObserved time.Time         `json:"firstObserved,omitempty"`
	LastObserved  *time.Time        `json:"lastObserved,omitempty"`
	Count         int               `json:"count,omitempty"`
	Message       string            `json:"message"`
	Hash          string            `json:"hash,omitempty"`
	Severity      string            `json:"severity,omitempty"`
	Source        string            `json:"source,omitempty"`
	Host          string            `json:"host,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

func (t *LogLine) SetHash() {
	t.Hash = tokenizer.Tokenize(t.Message)
}

func (t LogLine) GetDedupKey(fields ...string) string {
	if len(fields) == 0 {
		return ""
	}

	values := make([]string, len(fields))
	for i, field := range fields {
		values[i] = t.GetDedupField(field)
	}

	return strings.Join(values, "\u0000")
}

func (t LogLine) GetDedupField(field string) string {
	switch field {
	case "message":
		return fmt.Sprintf("msg::%s", t.Message)
	case "hash":
		return fmt.Sprintf("hash::%s", t.Hash)
	case "severity":
		return fmt.Sprintf("severity::%s", t.Severity)
	case "source":
		return fmt.Sprintf("source::%s", t.Source)
	case "host":
		return fmt.Sprintf("host::%s", t.Host)
	case "firstObserved":
		return fmt.Sprintf("firstObserved::%d", t.FirstObserved.UnixNano())
	case "lastObserved":
		if t.LastObserved == nil {
			return "lastObserved::unknown"
		}
		return fmt.Sprintf("lastObserved::%d", t.LastObserved.UnixNano())
	case "count":
		return fmt.Sprintf("count::%d", t.Count)
	case "id":
		return fmt.Sprintf("id::%s", t.ID)
	default:
		if t.Labels == nil {
			return fmt.Sprintf("label.%s=unknown", field)
		}

		if strings.HasPrefix(field, "label.") {
			return fmt.Sprintf("label.%s=%s", strings.TrimPrefix(field, "label."), t.Labels[strings.TrimPrefix(field, "label.")])
		}

		return ""
	}
}

func (t *LogLine) TemplateContext() map[string]any {
	return map[string]any{
		"id":            t.ID,
		"firstObserved": t.FirstObserved,
		"lastObserved":  t.LastObserved,
		"count":         t.Count,
		"message":       t.Message,
		"hash":          t.Hash,
		"severity":      t.Severity,
		"source":        t.Source,
		"host":          t.Host,
		"labels":        t.Labels,
	}
}

type LogResult struct {
	Metadata map[string]any `json:"metadata,omitempty"`
	Logs     []*LogLine     `json:"logs,omitempty"`
}

type LogsRequestBase struct {
	// The start time for the query
	// SupportsDatemath
	Start string `json:"start,omitempty"`

	// The end time for the query
	// Supports Datemath
	End string `json:"end,omitempty"`

	// Limit is the maximum number of lines to return
	Limit string `json:"limit,omitempty" template:"true"`
}

func (r *LogsRequestBase) GetStart() (time.Time, error) {
	return datemath.ParseAndEvaluate(r.Start, datemath.WithNow(time.Now()))
}

func (r *LogsRequestBase) GetEnd() (time.Time, error) {
	return datemath.ParseAndEvaluate(r.End, datemath.WithNow(time.Now()))
}
