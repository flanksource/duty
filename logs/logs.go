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

func (t LogLine) GetFieldKey(fields []string, messageFields ...string) string {
	if len(fields) == 0 {
		return ""
	}

	values := make([]string, len(fields))
	for i, field := range fields {
		values[i] = t.GetFieldValue(field, messageFields...)
	}

	return strings.Join(values, "\u0000")
}

var DefaultMessageFields = []string{"msg", "message"}

func (t LogLine) EffectiveMessage(messageFields ...string) string {
	if t.Message != "" {
		return t.Message
	}
	if t.Labels == nil {
		return ""
	}
	if len(messageFields) == 0 {
		messageFields = DefaultMessageFields
	}
	for _, field := range messageFields {
		if msg := t.Labels[field]; msg != "" {
			return msg
		}
	}
	return ""
}

func (t LogLine) GetFieldValue(field string, messageFields ...string) string {
	switch field {
	case "message":
		return fmt.Sprintf("msg::%s", t.EffectiveMessage(messageFields...))
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
		labelKey := field
		if strings.HasPrefix(field, "label.") {
			labelKey = strings.TrimPrefix(field, "label.")
		}

		if t.Labels == nil {
			return fmt.Sprintf("label.%s=unknown", labelKey)
		}

		return fmt.Sprintf("label.%s=%s", labelKey, t.Labels[labelKey])
	}
}

func (t *LogLine) TemplateContext(messageFields ...string) map[string]any {
	return map[string]any{
		"id":            t.ID,
		"firstObserved": t.FirstObserved,
		"lastObserved":  t.LastObserved,
		"count":         t.Count,
		"message":       t.EffectiveMessage(messageFields...),
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
	Groups   []*LogGroup    `json:"groups,omitempty"`
}

type LogGroup struct {
	Name   string            `json:"name,omitempty"`
	ID     string            `json:"id,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
	Logs   []*LogLine        `json:"logs,omitempty"`
}

func GroupLogs(result *LogResult, config FieldMappingConfig) {
	if len(result.Logs) == 0 {
		return
	}

	if len(config.GroupBy) == 0 {
		result.Logs = dedupLogs(result.Logs, config.DedupBy)
		return
	}

	groups := make(map[string][]*LogLine)
	var order []string
	for _, line := range result.Logs {
		key := line.GetFieldKey(config.GroupBy)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], line)
	}

	result.Groups = make([]*LogGroup, 0, len(groups))
	for _, key := range order {
		lines := groups[key]
		commonLabels := findCommonLabels(lines)

		for _, line := range lines {
			for k := range commonLabels {
				delete(line.Labels, k)
			}
		}

		result.Groups = append(result.Groups, &LogGroup{
			Labels: commonLabels,
			Logs:   dedupLogs(lines, config.DedupBy),
		})
	}

	result.Logs = nil
}

func dedupLogs(lines []*LogLine, dedupBy []string) []*LogLine {
	if len(dedupBy) == 0 || len(lines) == 0 {
		return lines
	}

	seen := make(map[string]*LogLine)
	var order []string
	for _, line := range lines {
		key := line.GetFieldKey(dedupBy)
		if existing, ok := seen[key]; ok {
			existing.Count += line.Count
			if line.FirstObserved.Before(existing.FirstObserved) {
				existing.FirstObserved = line.FirstObserved
			}
			if line.LastObserved != nil {
				if existing.LastObserved == nil || line.LastObserved.After(*existing.LastObserved) {
					existing.LastObserved = line.LastObserved
				}
			}
		} else {
			seen[key] = line
			order = append(order, key)
		}
	}

	result := make([]*LogLine, 0, len(order))
	for _, key := range order {
		result = append(result, seen[key])
	}
	return result
}

func findCommonLabels(lines []*LogLine) map[string]string {
	if len(lines) == 0 {
		return nil
	}

	common := make(map[string]string)
	for k, v := range lines[0].Labels {
		common[k] = v
	}

	for _, line := range lines[1:] {
		for k, v := range common {
			if lineVal, ok := line.Labels[k]; !ok || lineVal != v {
				delete(common, k)
			}
		}
		if len(common) == 0 {
			break
		}
	}

	return common
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
