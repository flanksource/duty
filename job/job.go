package job

import (
	gocontext "context"
	"fmt"
	"sync"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceTypeCheckStatuses = "check_statuses"
	ResourceTypeComponent     = "components"
	ResourceTypePlaybook      = "playbook"
	ResourceTypeUpstream      = "upstream"
)

var RetentionHour = Retention{
	Success:  1,
	Failed:   3,
	Age:      time.Hour,
	Interval: 5 * time.Minute,
}

var RetentionDay = Retention{
	Success:  3,
	Failed:   3,
	Age:      time.Hour,
	Interval: time.Hour * 24,
}

var Retention3Day = Retention{
	Success:  3,
	Failed:   3,
	Age:      time.Hour * 24 * 3,
	Interval: time.Hour * 4,
}

type Job struct {
	context.Context
	Name                     string
	Schedule                 string
	Singleton                bool
	Debug, Trace             bool
	Timeout                  time.Duration
	Fn                       func(ctx JobRuntime) error
	JobHistory               bool
	RunNow                   bool
	ID                       string
	ResourceID, ResourceType string
	entryID                  *cron.EntryID
	lock                     *sync.Mutex
	lastRun                  *time.Time
	Retention                Retention
	LastJob                  *models.JobHistory
	initialized              bool
	unschedule               func()
}

type Retention struct {
	Success, Failed int
	Age             time.Duration
	Interval        time.Duration
}

func (r Retention) String() string {
	return fmt.Sprintf("age=%s, interval=%s, success=%d, failed=%d", r.Age, r.Interval, r.Success, r.Failed)
}

type JobRuntime struct {
	context.Context
	Job       *Job
	Span      trace.Span
	History   *models.JobHistory
	Table, Id string
	runId     string
}

func (j *JobRuntime) ID() string {
	return fmt.Sprintf("[%s/%s]", j.Job.Name, j.runId)
}

func (j *JobRuntime) start() {
	j.Tracef("starting")
	j.History = models.NewJobHistory(j.Job.Name, "", "").Start()
	j.Job.LastJob = j.History
	if j.Job.ResourceID != "" {
		j.History.ResourceID = j.Job.ResourceID
	}
	if j.Job.ResourceType != "" {
		j.History.ResourceType = j.Job.ResourceType
	}
	if j.Job.JobHistory && j.Job.Retention.Success > 0 {
		if err := j.History.Persist(j.DB()); err != nil {
			j.Warnf("failed to persist history: %v", err)
		}
	}
}

func (j *JobRuntime) end() {
	j.History.End()
	if j.Job.JobHistory && (j.Job.Retention.Success > 0 || len(j.History.Errors) > 0) {
		if err := j.History.Persist(j.DB()); err != nil {
			j.Warnf("failed to persist history: %v", err)
		}
	}
}

func (j *JobRuntime) Failf(message string, args ...interface{}) {
	err := fmt.Sprintf(message, args...)
	j.Debugf(err)
	j.Span.SetStatus(codes.Error, err)
	if j.History != nil {
		j.History.AddError(err)
	}
}

func NewJob(ctx context.Context, name string, schedule string, fn func(ctx JobRuntime) error) *Job {
	return &Job{
		Context:    ctx,
		Retention:  Retention3Day,
		JobHistory: true,
		Name:       name,
		Schedule:   schedule,
		Fn:         fn,
	}
}

func (j *Job) SetTimeout(t time.Duration) *Job {
	j.Timeout = t
	return j
}

func (j *Job) FindHistory(statuses ...string) ([]models.JobHistory, error) {
	var items []models.JobHistory
	var err error
	if len(statuses) == 0 {
		err = j.DB().Where("name = ?", j.Name).Order("time_start DESC").Find(&items).Error
	} else {
		err = j.DB().Where("name = ? and status in ?", j.Name, statuses).Order("time_start DESC").Find(&items).Error
	}
	return items, err
}

func (j *Job) RunOnStart() *Job {
	j.RunNow = true
	return j
}

func (j *Job) Retain(r Retention) *Job {
	j.Retention = r
	return j
}

func (j *Job) SetID(id string) *Job {
	j.ID = id
	return j
}

func (j *Job) cleanupHistory() int {
	j.Context.Tracef("running cleanup: %v", j.Retention)
	db := j.Context.WithDBLogLevel("warn").DB()
	if err := db.Exec("DELETE FROM job_history WHERE name = ? AND now() - created_at >  interval '1 minute' * ?", j.Name, j.Retention.Age.Minutes()).Error; err != nil {
		j.Context.Warnf("failed to cleanup history %v", err)
	}
	query := `WITH ordered_history AS (
      SELECT
        id,
        status,
        ROW_NUMBER() OVER (PARTITION by resource_id, name, status ORDER BY created_at DESC)
      FROM job_history
			WHERE name = ? AND status IN ?
    )
    DELETE FROM job_history WHERE id IN (
      SELECT id FROM ordered_history WHERE row_number > ?
    )`

	policies := []struct {
		count    int
		statuses []string
	}{
		{j.Retention.Success, []string{models.StatusSuccess, models.StatusFinished}},
		{j.Retention.Failed, []string{models.StatusFailed, models.StatusWarning, models.StatusAborted}},
		{j.Retention.Success, []string{models.StatusRunning}},
	}
	count := 0
	for _, r := range policies {
		tx := db.Exec(query, j.Name, r.statuses, r.count)
		count += int(tx.RowsAffected)
		if tx.Error != nil {
			j.Context.Warnf("failed to cleanup history: %v", tx.Error)
		}
	}
	j.Context.Tracef("cleaned up %d records", count)
	return count
}

func (j *Job) Run() {
	j.init()
	if j.lastRun == nil || time.Since(*j.lastRun) > j.Retention.Interval {
		defer j.cleanupHistory()
	}
	j.lastRun = lo.ToPtr(time.Now())
	ctx, span := j.Context.StartSpan(j.Name)

	defer span.End()
	r := JobRuntime{
		Context: ctx,
		Span:    span,
		Job:     j,
	}
	if span.SpanContext().HasSpanID() {
		r.runId = span.SpanContext().SpanID().String()[0:8]
	} else {
		r.runId = uuid.NewString()[0:8]
	}

	r.start()
	defer r.end()
	if j.Singleton {
		ctx.Tracef("%s acquiring lock", r.ID())

		if j.lock == nil {
			j.lock = &sync.Mutex{}
		}
		if !j.lock.TryLock() {
			r.History.Status = models.StatusAborted
			ctx.Tracef("%s failed to acquire lock", r.ID())
			r.Failf("%s concurrent job aborted", r.ID())
			return
		}
		defer j.lock.Unlock()
	}

	if j.Timeout > 0 {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(j.Timeout)
		defer cancel()
	}

	err := j.Fn(r)
	ctx.Tracef("%s finished duration=%s, error=%s", r.ID(), time.Since(r.History.TimeStart), err)
	if err != nil {
		ctx.Error(err)
		r.History.AddError(err.Error())
	}
}

func getProperty(j *Job, properties map[string]string, property string) (string, bool) {
	if val, ok := properties[j.Name+"."+property]; ok {
		return val, ok
	}
	if val, ok := properties[fmt.Sprintf("%s[%s].%s", j.Name, j.ID, property)]; ok {
		return val, ok
	}
	return "", false
}

func (j *Job) init() {
	if j.initialized {
		return
	}
	properties := j.Context.Properties()

	if schedule, ok := getProperty(j, properties, "schedule"); ok {
		j.Schedule = schedule
	}

	if timeout, ok := getProperty(j, properties, "timeout"); ok {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			j.Context.Warnf("invalid timeout %s", timeout)
		}
		j.Timeout = duration
	}

	if history, ok := getProperty(j, properties, "history"); ok {
		j.JobHistory = !(history != "false")
	}

	if trace := properties["jobs.trace"]; trace == "true" {
		j.Trace = true
	} else if trace, ok := getProperty(j, properties, "trace"); ok {
		j.Trace = trace == "true"
	}

	if debug := properties["jobs.debug"]; debug == "true" {
		j.Debug = true
	} else if debug, ok := getProperty(j, properties, "debug"); ok {
		j.Debug = debug == "true"
	}

	if interval, ok := getProperty(j, properties, "retention.interval"); ok {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			j.Context.Warnf("invalid timeout %s", interval)
		}
		j.Retention.Interval = duration
	}

	if j.Retention.Interval.Nanoseconds() == 0 {
		j.Retention = Retention{
			Success: 3, Failed: 3,
			Interval: time.Hour * 1,
			Age:      time.Hour * 24 * 30,
		}
	}
	if j.Retention.Age.Nanoseconds() == 0 {
		j.Retention.Age = time.Hour * 24 * 7
	}
	if j.Retention.Interval.Nanoseconds() == 0 {
		j.Retention.Interval = time.Hour
	}
	if j.ID != "" {
		j.Context = j.Context.WithoutName().WithName(fmt.Sprintf("%s[%s]", j.Name, j.ID))
	} else {
		j.Context = j.Context.WithoutName().WithName(j.Name)
	}

	j.Context = j.Context.WithObject(v1.ObjectMeta{
		Name: j.Name,
		Annotations: map[string]string{
			"debug": lo.Ternary(j.Debug, "true", "false"),
			"trace": lo.Ternary(j.Trace, "true", "false"),
		},
	})

	if dbLevel, ok := getProperty(j, properties, "db-log-level"); ok {
		j.Context = j.Context.WithDBLogLevel(dbLevel)
	}

	j.Context.Tracef("initalized %v", j.String())
	j.initialized = true

}

func (j *Job) Label() string {
	if j.ID != "" {
		return fmt.Sprintf("%s/%s", j.Name, j.ID)
	}
	return j.Name
}

func (j *Job) String() string {
	return fmt.Sprintf("%s{schedule=%v, timeout=%v, history=%v, singleton=%v, retention=(%s)}",
		j.Name,
		j.Schedule,
		j.Timeout,
		j.JobHistory,
		j.Singleton,
		j.Retention,
	)
}

func (j *Job) AddToScheduler(cronRunner *cron.Cron) error {
	j.init()
	schedule := j.Schedule
	if override, ok := getProperty(j, j.Context.Properties(), "schedule"); ok {
		schedule = override
	}
	if schedule == "@never" {
		j.Context.Infof("skipping scheduling")
		return nil
	}
	j.Context.Infof("scheduled %s", schedule)
	entryID, err := cronRunner.AddJob(schedule, j)
	if err != nil {
		return fmt.Errorf("[%s] failed to schedule job: %s", j.Label(), err)
	}
	j.entryID = &entryID
	if j.RunNow {
		defer j.Run()
	}
	j.unschedule = func() {
		cronRunner.Remove(*j.entryID)
	}
	return nil
}

func (j *Job) GetEntry(cronRunner *cron.Cron) *cron.Entry {
	if j.entryID == nil {
		return nil
	}

	entry := cronRunner.Entry(*j.entryID)
	return &entry
}

func (j *Job) Unschedule() {
	if j.unschedule != nil {
		j.unschedule()
		j.unschedule = nil
	}
}

func (j *Job) Reschedule(schedule string, cronRunner *cron.Cron) error {
	if j.unschedule != nil {
		j.unschedule()
		j.unschedule = nil
	}
	j.Schedule = schedule
	return j.AddToScheduler(cronRunner)
}

func (j *Job) RemoveFromScheduler(cronRunner *cron.Cron) {
	if j.entryID == nil {
		return
	}
	cronRunner.Remove(*j.entryID)
}
