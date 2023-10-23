package job

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/robfig/cron/v3"
)

type Job struct {
	context.Context
	Name            string
	Schedule        string
	AllowConcurrent bool
	Timeout         time.Duration
	Fn              func(ctx JobRuntime, args ...any) error
	Args            []any
	RunNow          bool
	ID              string
	entryID         *cron.EntryID
}

type JobRuntime struct {
	context.Context
	Job     Job
	Started time.Time
	Ended   time.Time
}

func NewJob(ctx context.Context, name string, schedule string, fn func(ctx JobRuntime, args ...any) error, args ...any) *Job {
	return &Job{
		Context:  ctx,
		Name:     name,
		Schedule: schedule,
		Fn:       fn,
		Args:     args,
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
	ctx.Debugf("Running job: %s", j.Name)

	if j.Timeout > 0 {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(j.Timeout)
		defer cancel()
	}

	r := JobRuntime{
		Context: ctx,
		Job:     j,
	}

	defer span.End()
	if err := j.Fn(r, j.Args...); err != nil {
		ctx.Error(err)
	}
}

func (j *Job) AddToScheduler(cronRunner *cron.Cron) error {
	entryID, err := cronRunner.AddJob(j.Schedule, j)
	if err != nil {
		return fmt.Errorf("failed to schedule job: %s", j.Name)
	}
	j.entryID = &entryID
	if j.RunNow {
		j.Run()
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

func (j Job) RemoveFromScheduler(cronRunner *cron.Cron) *cron.Entry {
	for _, entry := range cronRunner.Entries() {
		if entry.Job.(Job).Name == j.Name {
			return &entry
		}
	}
	return nil
}
