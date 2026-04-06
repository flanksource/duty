package query

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	clickyapi "github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
)

type QueryLogger struct {
	logger logger.Verbose
}

type QueryTimer struct {
	logger  logger.Verbose
	label   clickyapi.Text
	start   time.Time
	results any
	ended   bool
}

func NewQueryLogger(ctx context.Context) QueryLogger {
	l := ctx.Logger.V(3)
	if ctx.Properties().On(false, "query.log") {
		l = ctx.Logger.V(0)
	}
	return QueryLogger{logger: l}
}

func (q QueryLogger) Start(entity string) *QueryTimer {
	return &QueryTimer{
		logger: q.logger,
		label:  clickyapi.Text{Content: "[" + entity + "]", Style: "text-blue-600 font-bold"},
		start:  time.Now(),
	}
}

func (t *QueryTimer) Arg(key string, value any) *QueryTimer {
	s := fmt.Sprintf("%v", value)
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	t.label = t.label.AddText(fmt.Sprintf(" %s=", key), "text-gray-500").
		AddText(s)
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
	if !t.logger.Enabled() {
		return
	}

	elapsed := time.Since(t.start)
	label := t.label.AddText(" => ", "text-gray-400")

	if err != nil && *err != nil {
		label = label.AddText(fmt.Sprintf("error: %v", *err), "text-red-600")
	} else if t.results != nil {
		count := sliceLen(t.results)
		countStyle := "text-green-600"
		if count == 0 {
			countStyle = "text-red-600"
		}
		label = label.AddText(fmt.Sprintf("%d", count), countStyle)
		label = label.AddText(summaryText(t.results, count), "text-gray-400")
	} else {
		label = label.AddText("timed out", "text-yellow-600")
	}

	label = label.AddText(fmt.Sprintf(" in %dms", elapsed.Milliseconds()), "text-gray-400")
	t.logger.Infof("%s", label.ANSI())
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
