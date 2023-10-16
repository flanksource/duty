package job

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/robfig/cron/v3"
)

type job struct {
	context.Context
	Name            string
	Schedule        string
	AllowConcurrent bool
	Timeout         time.Duration
	Fn              func(ctx JobRuntime) error
	RunNow          bool
}

type JobRuntime struct {
	context.Context
	Job     job
	Started time.Time
	Ended   time.Time
}

func NewJob(ctx context.Context, name string, schedule string, fn func(ctx JobRuntime) error) *job {
	return &job{
		Context:  ctx,
		Name:     name,
		Schedule: schedule,
		Fn:       fn,
	}
}

func (j *job) SetTimeout(t time.Duration) *job {
	j.Timeout = t
	return j
}

func (j *job) RunOnStart() *job {
	j.RunNow = true
	return j
}

func (j job) Run() {
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
	if err := j.Fn(r); err != nil {
		ctx.Error(err)
	}
}

func (j job) AddToScheduler(cronRunner *cron.Cron) error {
	if _, err := cronRunner.AddJob(j.Schedule, j); err != nil {
		return fmt.Errorf("failed to schedule job: %s", j.Name)
	}
	if j.RunNow {
		j.Run()
	}
	return nil
}
