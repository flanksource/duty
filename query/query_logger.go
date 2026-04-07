package query

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	clickyapi "github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
)

type QueryLogEntry struct {
	Name     string `json:"name"`
	Args     string `json:"args,omitempty"`
	Count    int    `json:"count"`
	Duration int64  `json:"duration"`
	Error    string `json:"error,omitempty"`
	Summary  string `json:"summary,omitempty"`
}

type QueryLog struct {
	mu      sync.Mutex
	entries []QueryLogEntry
}

func (q *QueryLog) Append(e QueryLogEntry) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, e)
}

func (q *QueryLog) Entries() []QueryLogEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	return slices.Clone(q.entries)
}

type queryLogKey struct{}

func WithQueryLog(ctx context.Context) (context.Context, *QueryLog) {
	log := &QueryLog{}
	return ctx.WithValue(queryLogKey{}, log), log
}

func GetQueryLog(ctx context.Context) *QueryLog {
	if v, ok := ctx.Value(queryLogKey{}).(*QueryLog); ok {
		return v
	}
	return nil
}

type QueryLogger struct {
	logger   logger.Verbose
	queryLog *QueryLog
}

type QueryTimer struct {
	logger   logger.Verbose
	queryLog *QueryLog
	name     string
	args     string
	label    clickyapi.Text
	start    time.Time
	results  any
	ended    bool
}

func NewQueryLogger(ctx context.Context) QueryLogger {
	l := ctx.Logger.V(3)
	if ctx.Properties().On(false, "query.log") {
		l = ctx.Logger.V(0)
	}
	return QueryLogger{logger: l, queryLog: GetQueryLog(ctx)}
}

func (q QueryLogger) Start(entity string) *QueryTimer {
	return &QueryTimer{
		logger:   q.logger,
		queryLog: q.queryLog,
		name:     entity,
		label:    clickyapi.Text{Content: "[" + entity + "]", Style: "text-blue-600 font-bold"},
		start:    time.Now(),
	}
}

func (t *QueryTimer) Arg(key string, value any) *QueryTimer {
	s := fmt.Sprintf("%v", value)
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	t.label = t.label.AddText(fmt.Sprintf(" %s=", key), "text-gray-500").
		AddText(s)
	if t.args != "" {
		t.args += " "
	}
	t.args += fmt.Sprintf("%s=%s", key, s)
	return t
}

func (t *QueryTimer) Results(results any) *QueryTimer {
	t.results = results
	return t
}

func (t *QueryTimer) End(err *error) {
	if t.ended {
		return
	}
	t.ended = true

	elapsed := time.Since(t.start)

	var entry QueryLogEntry
	if t.queryLog != nil {
		entry.Name = t.name
		entry.Args = t.args
		entry.Duration = elapsed.Milliseconds()
	}

	label := t.label.AddText(" => ", "text-gray-400")

	if err != nil && *err != nil {
		label = label.AddText(fmt.Sprintf("error: %v", *err), "text-red-600")
		if t.queryLog != nil {
			entry.Error = (*err).Error()
		}
	} else if t.results != nil {
		count := sliceLen(t.results)
		countStyle := "text-green-600"
		if count == 0 {
			countStyle = "text-red-600"
		}
		label = label.AddText(fmt.Sprintf("%d", count), countStyle)
		summary := summaryText(t.results, count)
		label = label.AddText(summary, "text-gray-400")
		if t.queryLog != nil {
			entry.Count = count
			entry.Summary = summary
		}
	} else {
		label = label.AddText("timed out", "text-yellow-600")
	}

	label = label.AddText(fmt.Sprintf(" in %dms", elapsed.Milliseconds()), "text-gray-400")

	if t.queryLog != nil {
		t.queryLog.Append(entry)
	}

	if t.logger.Enabled() {
		t.logger.Infof("%s", label.ANSI())
	}
}

func sliceLen(v any) int {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		return rv.Len()
	}
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		return 1
	}
	return 0
}

func summaryText(v any, count int) string {
	if count == 0 {
		return ""
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return ""
	}

	// Try grouped summary first (e.g. "CodeDeployment: 5, BackupCompleted: 3")
	if grouped := groupedSummary(rv); grouped != "" {
		return " [" + grouped + "]"
	}

	const maxInline = 2
	shown := count
	if shown > maxInline {
		shown = maxInline
	}

	var parts []string
	for i := 0; i < shown; i++ {
		parts = append(parts, itemLogSummary(rv.Index(i).Interface()))
	}
	summary := strings.Join(parts, ", ")
	if count > maxInline {
		summary += fmt.Sprintf(", ...%d more", count-maxInline)
	}
	return " [" + summary + "]"
}

func groupedSummary(rv reflect.Value) string {
	if rv.Len() == 0 {
		return ""
	}
	first := rv.Index(0).Interface()
	if _, ok := first.(QueryLogSummary); !ok {
		return ""
	}
	counts := make(map[string]int)
	var order []string
	for i := 0; i < rv.Len(); i++ {
		s := rv.Index(i).Interface().(QueryLogSummary).QueryLogSummary()
		if counts[s] == 0 {
			order = append(order, s)
		}
		counts[s]++
	}
	var parts []string
	for _, key := range order {
		parts = append(parts, fmt.Sprintf("%s: %d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}
