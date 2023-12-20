package job

import (
	gocontext "context"
	"fmt"
	"sync"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Job struct {
	context.Context
	Name       string
	Schedule   string
	Singleton  bool
	Timeout    time.Duration
	Fn         func(ctx JobRuntime) error
	JobHistory bool
	RunNow     bool
	ID         string
	entryID    *cron.EntryID
	lock       *sync.Mutex
	unschedule func()
}

type JobRuntime struct {
	context.Context
	Job       Job
	Span      trace.Span
	History   *models.JobHistory
	Started   time.Time
	Ended     time.Time
	Table, Id string
}

func (j *JobRuntime) start() {
	j.History = models.NewJobHistory(j.Job.Name, "", "").Start()

	if j.Job.JobHistory {
		_ = j.History.Persist(j.DB())
	}
}

func (j *JobRuntime) end() {
	if j.Job.JobHistory {
		_ = j.History.Persist(j.DB())
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

func (j *Job) RunOnStart() *Job {
	j.RunNow = true
	return j
}

func (j *Job) SetID(id string) *Job {
	j.ID = id
	return j
}

func (j Job) Run() {
	ctx, span := j.StartSpan(j.Name)
	defer span.End()
	r := JobRuntime{
		Context: ctx,
		Job:     j,
	}

	r.start()
	defer r.end()
	if !j.Singleton {
		if j.lock == nil {
			j.lock = &sync.Mutex{}
		}
		if !j.lock.TryLock() {
			r.Failf("Concurrent job of %s aborted", j.Name)
			return
		}
		defer j.lock.Unlock()
	}

	ctx.Debugf("Running job: %s", j.Name)

	if j.Timeout > 0 {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(j.Timeout)
		defer cancel()
	}

	if err := j.Fn(r); err != nil {
		ctx.Error(err)
	}
}

func (j *Job) AddToScheduler(cronRunner *cron.Cron) error {
	properties := j.Context.Properties()
	if schedule, ok := properties[j.Name+".schedule"]; ok {
		if schedule == "@never" {
			logger.Infof("Skipping scheduling of %s", j.Name)
			return nil
		}
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

func (j Job) GetEntry(cronRunner *cron.Cron) *cron.Entry {
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

func (j Job) RemoveFromScheduler(cronRunner *cron.Cron) {
	if j.entryID == nil {
		return
	}
	cronRunner.Remove(*j.entryID)
}
