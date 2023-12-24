package job

import (
	gocontext "context"
	"fmt"
	"sync"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Job struct {
	context.Context
	Name         string
	Schedule     string
	Singleton    bool
	Debug, Trace bool
	Timeout      time.Duration
	Fn           func(ctx JobRuntime) error
	JobHistory   bool
	RunNow       bool
	ID           string
	entryID      *cron.EntryID
	lock         *sync.Mutex
	lastRun      *time.Time
	Retention    Retention
	initialized  bool
	unschedule   func()
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
	j.Context.Debugf("%s starting", j.ID())
	j.History = models.NewJobHistory(j.Job.Name, "", "").Start()

	if j.Job.JobHistory {
		if err := j.History.Persist(j.DB()); err != nil {
			logger.Warnf("%s failed to persist history: %v", j.ID(), err)
		}
	}
}

func (j *JobRuntime) end() {
	if j.Job.JobHistory {
		if err := j.History.Persist(j.DB()); err != nil {
			logger.Warnf("%s failed to persist history: %v", j.ID(), err)
		}
	}
}

func (j *JobRuntime) Failf(message string, args ...interface{}) {
	err := fmt.Sprintf(message, args...)
	j.Context.Debugf(err)
	j.Span.SetStatus(codes.Error, err)
	if j.History != nil {
		j.History.AddError(err)
	}
}

func NewJob(ctx context.Context, name string, schedule string, fn func(ctx JobRuntime) error) *Job {
	return &Job{
		Context:  ctx,
		Name:     name,
		Schedule: schedule,
		Fn:       fn,
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
		err = j.DB().Where("name = ?", j.Name).Find(&items).Error
	} else {
		err = j.DB().Where("name = ? and status in ?", j.Name, statuses).Find(&items).Error
	}
	return items, err
}

func (j *Job) RunOnStart() *Job {
	j.RunNow = true
	return j
}

func (j *Job) SetID(id string) *Job {
	j.ID = id
	return j
}

func (j *Job) cleanupHistory() int {
	j.Context.Debugf("[%s] running cleanup: %v", j.Name, j.Retention)
	db := j.Context.DB()
	if err := db.Exec("DELETE FROM job_history WHERE name = ? AND now() - created_at >  interval '1 minute' * ?", j.Name, j.Retention.Age.Minutes()).Error; err != nil {
		logger.Warnf("Failed to cleanup history for : %s", j.Name)
	}
	query := `
    WITH ordered_history AS (
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
		{j.Retention.Failed, []string{models.StatusFailed, models.StatusWarning}},
		{j.Retention.Success, []string{models.StatusRunning}},
	}
	count := 0
	for _, r := range policies {
		tx := db.Exec(query, j.Name, r.statuses, r.count)
		count += int(tx.RowsAffected)
		if tx.Error != nil {
			logger.Warnf("Failed to cleanup history for %s: %v", j.Name, tx.Error)
		}
	}
	j.Context.Debugf("[%s] cleaned up %d records", j.Name, count)
	return count
}

func (j *Job) Run() {
	j.init()
	if j.lastRun != nil && time.Since(*j.lastRun) > j.Retention.Interval {
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

	if j.Singleton {
		ctx.Tracef("%s acquiring lock", r.ID())

		if j.lock == nil {
			j.lock = &sync.Mutex{}
		}
		if !j.lock.TryLock() {
			ctx.Tracef("%s failed to acquire lock", r.ID())
			r.Failf("%s concurrent job aborted", r.ID())
			return
		}
		defer j.lock.Unlock()
	}

	defer r.end()

	if j.Timeout > 0 {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(j.Timeout)
		defer cancel()
	}

	err := j.Fn(r)
	ctx.Tracef("%s finished duration=%s, error=%s", r.ID(), time.Since(r.History.TimeStart), err)
	if err != nil {
		ctx.Error(err)
	}
}

func (j *Job) init() {
	if j.initialized {
		return
	}
	properties := j.Context.Properties()
	if schedule, ok := properties[j.Name+".schedule"]; ok {
		j.Schedule = schedule
	}

	if timeout, ok := properties[j.Name+".timeout"]; ok {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			logger.Warnf("Invalid timeout for %s: %s", j.Name, timeout)
		}
		j.Timeout = duration
	}

	if history, ok := properties[j.Name+".history"]; ok {
		j.JobHistory = !(history != "false")
	}

	if debug, ok := properties[j.Name+".debug"]; ok {
		j.Debug = debug == "true"
	}

	if trace, ok := properties[j.Name+".trace"]; ok {
		j.Trace = trace == "true"
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

	j.Context = j.Context.WithObject(v1.ObjectMeta{
		Name: j.Name,
		Annotations: map[string]string{
			"debug": lo.Ternary(j.Debug, "true", "false"),
			"trace": lo.Ternary(j.Trace, "true", "false"),
		},
	})

	if dbLevel, ok := properties[j.Name+".db.level"]; ok {
		j.Context = j.Context.WithDBLogLevel(dbLevel)
	}

	j.Context.Debugf("initalized: %v", j)
	j.initialized = true

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
	if j.Schedule == "@never" {
		logger.Infof("Skipping scheduling of %s", j.Name)
		return nil
	}
	entryID, err := cronRunner.AddJob(j.Schedule, j)
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %s", j.Name, err)
	}
	j.entryID = &entryID
	if j.RunNow {
		j.Run()
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

func (j *Job) RemoveFromScheduler(cronRunner *cron.Cron) {
	if j.entryID == nil {
		return
	}
	cronRunner.Remove(*j.entryID)
}
