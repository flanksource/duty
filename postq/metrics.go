package postq

import (
	"slices"
	"strings"

	"github.com/flanksource/duty/context"
)

const preHandlerErrorsMetricName = "postq_prehandler_errors_total"

func recordPreHandlerError(ctx context.Context, consumer string, watchEvents []string, err error) {
	ctx.Counter(
		preHandlerErrorsMetricName,
		"consumer", consumer,
		"event", eventMetricLabel(watchEvents),
		"kind", classifyPreHandlerError(err),
	).Add(1)
}

func eventMetricLabel(events []string) string {
	if len(events) == 0 {
		return "unknown"
	}

	if len(events) == 1 {
		return events[0]
	}

	copyEvents := append([]string(nil), events...)
	slices.Sort(copyEvents)
	return strings.Join(copyEvents, "|")
}

func classifyPreHandlerError(err error) string {
	if err == nil {
		return "unknown"
	}

	e := strings.ToLower(err.Error())

	switch {
	case strings.Contains(e, "scan error"),
		strings.Contains(e, "cannot unmarshal"),
		strings.Contains(e, "unsupported scan"):
		return "scan_decode"
	case strings.Contains(e, "error starting transaction"):
		return "tx_begin"
	case strings.Contains(e, "error fetching events"):
		return "fetch"
	default:
		return "other"
	}
}
