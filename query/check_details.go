package query

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/flanksource/commons/duration"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
	"gonum.org/v1/gonum/stat"
)

// Default search window
var DefaultCheckQueryWindow = "1h"

type Timeseries struct {
	Key      string `json:"key,omitempty"`
	Time     string `json:"time,omitempty"`
	Status   bool   `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration int    `json:"duration"`
	// Count is the number of times the check has been run in the specified time window
	Count  int `json:"count,omitempty"`
	Passed int `json:"passed,omitempty"`
	Failed int `json:"failed,omitempty"`
}

type CheckQueryParams struct {
	Check           string
	CanaryID        string
	Start, End      string
	IncludeMessages bool
	IncludeDetails  bool
	_start, _end    *time.Time
	StatusCount     int
	Labels          map[string]string
	Trace           bool
	WindowDuration  time.Duration
}

func (q *CheckQueryParams) Validate() error {
	start, err := timeV(q.Start)
	if err != nil {
		return fmt.Errorf("start is invalid: %w", err)
	}
	end, err := timeV(q.End)
	if err != nil {
		return fmt.Errorf("end is invalid: %w", err)
	}
	if start != nil && end != nil {
		if end.Before(*start) {
			return fmt.Errorf("end time must be after start time")
		}
	}

	return nil
}

func (q CheckQueryParams) GetStartTime() *time.Time {
	if q._start != nil {
		return q._start
	}
	if q.Start == "" {
		q._start = lo.ToPtr(time.Now().Add(-q.WindowDuration))
	} else {
		q._start, _ = timeV(q.Start)
	}
	return q._start
}

func (q CheckQueryParams) GetEndTime() *time.Time {
	if q._end != nil {
		return q._end
	}
	if q.End == "" {
		q._end = lo.ToPtr(time.Now())
	} else {
		q._end, _ = timeV(q.End)
	}
	return q._end
}

func (q CheckQueryParams) GetWhereClause() (string, map[string]interface{}, error) {
	clause := ""
	args := make(map[string]interface{})
	and := " AND "
	if q.Check != "" {
		clause = "check_id = :check_key"
		args["check_key"] = q.Check
	}
	if q.Start != "" && q.End == "" {
		if clause != "" {
			clause += and
		}
		start, arg, err := parseDuration(q.Start, "start")
		if err != nil {
			return "", nil, err
		}
		args["start"] = arg
		clause += "time > " + start
	} else if q.Start == "" && q.End != "" {
		if clause != "" {
			clause += and
		}
		end, arg, err := parseDuration(q.End, "end")
		if err != nil {
			return "", nil, err
		}
		args["end"] = arg
		clause += "time < " + end
	}
	if q.Start != "" && q.End != "" {
		if clause != "" {
			clause += and
		}
		start, arg, err := parseDuration(q.Start, "start")
		if err != nil {
			return "", nil, err
		}
		args["start"] = arg
		end, arg, err := parseDuration(q.End, "end")
		if err != nil {
			return "", nil, err
		}
		args["end"] = arg
		clause += "time BETWEEN " + start + and + end
	}
	return strings.TrimSpace(clause), args, nil
}

func (q CheckQueryParams) ExecuteDetails(ctx context.Context) ([]Timeseries, types.Uptime, types.Latency, error) {
	start := q.GetStartTime().Format(time.RFC3339)
	end := q.GetEndTime().Format(time.RFC3339)

	query := `
WITH grouped_by_window AS (
	SELECT
		duration,
		status,
		CASE  WHEN check_statuses.status = TRUE THEN 1  ELSE 0 END AS passed,
		CASE  WHEN check_statuses.status = FALSE THEN 1  ELSE 0 END AS failed,
		to_timestamp(floor((extract(epoch FROM time) + $1) / $2) * $2) AS time
	FROM check_statuses
	WHERE
		time >= $3 AND
		time <= $4 AND
		check_id = $5
)
SELECT
  time,
  bool_and(status),
  AVG(duration)::integer as duration,
	sum(passed) as passed,
	sum(failed) as failed
FROM
  grouped_by_window
GROUP BY time
ORDER BY time
`
	args := []any{q.WindowDuration.Seconds() / 2, q.WindowDuration.Seconds(), start, end, q.Check}

	if q.WindowDuration == 0 {
		// FIXME
		query = `SELECT time, status, duration,
		CASE  WHEN check_statuses.status = TRUE THEN 1  ELSE 0 END AS passed,
		CASE  WHEN check_statuses.status = FALSE THEN 1  ELSE 0 END AS failed
		FROM check_statuses WHERE time >= $1 AND time <= $2 AND check_id = $3`
		args = []any{start, end, q.Check}
	}
	uptime := types.Uptime{}
	var latencies []float64

	rows, err := ctx.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, uptime, types.Latency{}, err
	}
	defer rows.Close()

	var results []Timeseries
	for rows.Next() {
		var datapoint Timeseries
		var ts time.Time
		if err := rows.Scan(&ts, &datapoint.Status, &datapoint.Duration, &datapoint.Passed, &datapoint.Failed); err != nil {
			return nil, uptime, types.Latency{}, err
		}
		uptime.Failed += datapoint.Failed
		uptime.Passed += datapoint.Passed
		latencies = append(latencies, float64(datapoint.Duration))
		datapoint.Time = ts.Format(time.RFC3339)
		results = append(results, datapoint)
	}

	// stat.Quantile panics on empty lists so we return early
	if len(results) == 0 {
		return nil, uptime, types.Latency{}, nil
	}

	// Sorting is required before calculating latencies else Quantile panics
	slices.Sort(latencies)
	latency := types.Latency{
		Percentile99: stat.Quantile(0.99, stat.Empirical, latencies, nil),
		Percentile97: stat.Quantile(0.97, stat.Empirical, latencies, nil),
		Percentile95: stat.Quantile(0.95, stat.Empirical, latencies, nil),
	}

	return results, uptime, latency, nil
}

func (q CheckQueryParams) String() string {
	return fmt.Sprintf("check:=%s, start=%s, end=%s", q.Check, q.Start, q.End)
}

func (q *CheckQueryParams) Init(queryParams url.Values) error {
	since := queryParams.Get("since")
	if since == "" {
		since = queryParams.Get("start")
	}
	if since == "" {
		since = DefaultCheckQueryWindow
	}

	until := queryParams.Get("until")
	if until == "" {
		until = queryParams.Get("end")
	}
	if until == "" {
		until = "0s"
	}

	*q = CheckQueryParams{
		Start:           since,
		End:             until,
		IncludeMessages: isTrue(queryParams.Get("includeMessages")),
		IncludeDetails:  isTrue(queryParams.Get("includeDetails")),
		Check:           queryParams.Get("check"),
		Trace:           isTrue(queryParams.Get("trace")),
		CanaryID:        queryParams.Get("canary_id"),
	}

	timeRange := q.GetEndTime().Sub(*q.GetStartTime())
	if timeRange <= time.Hour*2 {
		q.WindowDuration = time.Minute
	} else if timeRange <= time.Hour*24 {
		q.WindowDuration = time.Minute * 15
	} else if timeRange <= time.Hour*24*7 {
		q.WindowDuration = time.Minute * 60
	} else {
		q.WindowDuration = time.Hour * 4
	}

	return q.Validate()
}

func isTrue(v string) bool {
	return v == "true"
}

func timeV(v interface{}) (*time.Time, error) {
	if v == nil {
		return nil, nil
	}
	switch v := v.(type) {
	case time.Time:
		return &v, nil
	case time.Duration:
		t := time.Now().Add(v * -1)
		return &t, nil
	case string:
		if v == "" {
			return nil, nil
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return &t, nil
		} else if d, err := duration.ParseDuration(v); err == nil {
			t := time.Now().Add(time.Duration(d) * -1)
			return &t, nil
		}
		return nil, fmt.Errorf("time must be a duration or RFC3339 timestamp")
	}
	return nil, fmt.Errorf("unknown time type %T", v)
}

func parseDuration(d string, name string) (clause string, arg interface{}, err error) {
	if d == "" {
		return "", nil, nil
	}
	dur, err := duration.ParseDuration(d)
	if err == nil {
		return fmt.Sprintf("(NOW() at TIME ZONE 'utc' - Interval '1 minute' * :%s)", name), dur.Minutes(), nil
	}
	if timestamp, err := time.Parse(time.RFC3339, d); err == nil {
		return ":" + name, timestamp, nil
	}
	return "", nil, fmt.Errorf("start time must be a duration or RFC3339 timestamp")
}
