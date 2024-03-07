package job

import (
	"container/ring"
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
)

const (
	ResourceTypeCheckStatuses = "check_statuses"
	ResourceTypeComponent     = "components"
	ResourceTypePlaybook      = "playbook"
	ResourceTypeScraper       = "config_scraper"
	ResourceTypeUpstream      = "upstream"
)

var evictedJobs chan uuid.UUID

// deleteEvictedJobs deletes job_history rows from the DB every job.eviction.period(1m),
// jobs send rows to be deleted by maintaining a circular buffer by status type
func deleteEvictedJobs(ctx context.Context) {
	period := ctx.Properties().Duration("job.eviction.period", time.Minute)
	ctx.Infof("Cleaning up jobs every %v", period)
	for {
		items, _, _, _ := lo.BufferWithTimeout(evictedJobs, 32, 5*time.Second)
		if len(items) == 0 {
			time.Sleep(period)
			continue
		}
		if tx := ctx.DB().Exec("DELETE FROM job_history WHERE id in ?", items); tx.Error != nil {
			ctx.Errorf("Failed to delete job entries: %v", tx.Error)
			time.Sleep(1 * time.Minute)
		} else {
			ctx.Tracef("Deleted %d job history items", tx.RowsAffected)
		}
	}
}

var RetentionMinutes = Retention{
	Success: 1,
	Failed:  3,
	Age:     time.Minute * 15,
}

var RetentionHour = Retention{
	Success: 1,
	Failed:  3,
	Age:     time.Hour,
}

var RetentionFailed = Retention{
	Success: 0,
	Failed:  1,
	Age:     time.Hour * 24 * 2,
}

var RetentionShort = Retention{
	Success: 1,
	Failed:  1,
	Age:     time.Hour,
}

var RetentionDay = Retention{
	Success: 3,
	Failed:  3,
	Age:     time.Hour * 24,
}

var Retention3Day = Retention{
	Success: 3,
	Failed:  3,
	Age:     time.Hour * 24 * 3,
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
	lastHistoryCleanup       time.Time
	Retention                Retention
	LastJob                  *models.JobHistory
	initialized              bool
	unschedule               func()
	statusRing               StatusRing
}

type StatusRing struct {
	lock      sync.Mutex
	rings     map[string]*ring.Ring
	evicted   chan uuid.UUID
	retention Retention
}

func newStatusRing(r Retention, evicted chan uuid.UUID) StatusRing {
	return StatusRing{
		lock:      sync.Mutex{},
		retention: r,
		rings:     make(map[string]*ring.Ring),
		evicted:   evicted,
	}
}

func (sr *StatusRing) Add(job *models.JobHistory) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	var r *ring.Ring
	var ok bool
	if r, ok = sr.rings[job.Status]; !ok {
		r = ring.New(sr.retention.Count(job.Status) + 1)
		sr.rings[job.Status] = r
	}
	r.Value = job.ID
	r = r.Next()

	if r.Value != nil {
		sr.evicted <- r.Value.(uuid.UUID)
	}
	sr.rings[job.Status] = r
}

type Retention struct {
	// Success is the number of finished/success job history to retain
	Success int

	// Failed is the number of unsuccessful job history to retain
	Failed int

	// Age is the maximum age of job history to retain
	Age time.Duration

	// Interval for job history cleanup
	Interval time.Duration

	// Data ...?
	Data bool
}

func (r Retention) Count(status string) int {
	if status == models.StatusAborted || status == models.StatusFailed || status == models.StatusWarning {
		return r.Failed
	}
	return r.Success
}

func (r Retention) WithData() Retention {
	r.Data = true
	return r
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
	j.Context.Counter("job_started", "name", j.Job.Name).Add(1)
	j.History = models.NewJobHistory(j.Logger, j.Job.Name, "", "").Start()
	j.Job.LastJob = j.History
	if j.Job.ResourceID != "" {
		j.History.ResourceID = j.Job.ResourceID
	}
	if j.Job.ResourceType != "" {
		j.History.ResourceType = j.Job.ResourceType
	}
	if j.Job.JobHistory && j.Job.Retention.Success > 0 {
		if err := j.History.Persist(j.FastDB()); err != nil {
			j.Warnf("failed to persist history: %v", err)
		}
	}
}

func (j *JobRuntime) end() {
	j.Context.Counter("job", "name", j.Job.Name, "id", j.Job.ResourceID, "resource", j.Job.ResourceType, "status", j.History.Status).Add(1)
	j.Context.Histogram("job_duration", "name", j.Job.Name, "id", j.Job.ResourceID, "resource", j.Job.ResourceType, "status", j.History.Status).Since(j.History.TimeStart)

	j.History.End()
	if j.Job.JobHistory && (j.Job.Retention.Success > 0 || len(j.History.Errors) > 0) {
		if err := j.History.Persist(j.FastDB()); err != nil {
			j.Warnf("failed to persist history: %v", err)
		}
	}
	j.Job.statusRing.Add(j.History)
}

func (j *JobRuntime) Failf(message string, args ...interface{}) {
	err := fmt.Sprintf(message, args...)
	j.Logger.WithSkipReportLevel(1).Debugf(err)
	j.Span.SetStatus(codes.Error, err)
	if j.History != nil {
		j.History.AddErrorWithSkipReportLevel(err, 1)
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
	j.Context.Logger.V(4).Infof("running cleanup: %v", j.Retention)
	ctx, span := j.Context.StartSpan("CleanupHistory")
	defer span.End()
	db := ctx.FastDB()
	if err := db.Exec("DELETE FROM job_history WHERE name = ? AND resource_id = ? AND now() - created_at >  interval '1 minute' * ?", j.Name, j.ResourceID, j.Retention.Age.Minutes()).Error; err != nil {
		ctx.Warnf("failed to cleanup history %v", err)
	}
	query := `WITH ordered_history AS (
      SELECT
        id,
        status,
        ROW_NUMBER() OVER (PARTITION by resource_id, name, status ORDER BY created_at DESC)
      FROM job_history
			WHERE name = ? AND resource_id = ? AND status IN ?
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
		tx := db.Exec(query, j.Name, j.ResourceID, r.statuses, r.count)
		count += int(tx.RowsAffected)
		if tx.Error != nil {
			ctx.Warnf("failed to cleanup history: %v", tx.Error)
		}
	}
	ctx.Logger.V(3).Infof("cleaned up %d records", count)
	return count
}

func (j *Job) Run() {
	j.init()
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
		ctx.Logger.V(4).Infof("acquiring lock")

		if j.lock == nil {
			j.lock = &sync.Mutex{}
		}
		if !j.lock.TryLock() {
			r.History.Status = models.StatusAborted
			ctx.Tracef("failed to acquire lock")
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

	if shouldCleanupHistory(j.lastHistoryCleanup, j.Retention.Age) {
		defer func() {
			j.cleanupHistory()
			j.lastHistoryCleanup = time.Now()
		}()
	}

	err := j.Fn(r)
	if err != nil {
		ctx.Tracef("finished duration=%s, error=%s", time.Since(r.History.TimeStart), err)
		r.History.AddErrorWithSkipReportLevel(err.Error(), 1)
	} else {
		ctx.Tracef("finished duration=%s", time.Since(r.History.TimeStart))
	}
}

func shouldCleanupHistory(lastCleanup time.Time, retentionAge time.Duration) bool {
	cleanupInterval := time.Hour * 6

	// If retention is more than a day, cleanup every half a day
	if retentionAge >= (24 * time.Hour) {
		cleanupInterval = 12 * time.Hour
	}

	// If retention is less than an hour, cleanup every hour
	if retentionAge <= (time.Hour) {
		cleanupInterval = time.Hour
	}

	return time.Since(lastCleanup) >= cleanupInterval
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
	if evictedJobs == nil {
		evictedJobs = make(chan uuid.UUID, 1000)
		go deleteEvictedJobs(j.Context)
	}
	if j.initialized {
		return
	}

	j.lastHistoryCleanup = time.Now()

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

	// Set default retention if it is unset
	if j.Retention.Age.Nanoseconds() == 0 {
		j.Retention = Retention{
			Success: 1, Failed: 3,
			Age: time.Hour * 24,
		}
	}

	if interval, ok := getProperty(j, properties, "retention.interval"); ok {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			j.Context.Warnf("invalid timeout %s", interval)
		}
		j.Retention.Interval = duration
	} else {
		j.Retention.Interval = 4 * time.Hour
	}

	j.statusRing = newStatusRing(j.Retention, evictedJobs)

	if j.ID != "" {
		j.Context = j.Context.WithoutName().WithName(fmt.Sprintf("%s[%s]", j.Name, j.ID))
	} else {
		j.Context = j.Context.WithoutName().WithName(j.Name)
	}

	obj := j.Context.GetObjectMeta()
	if obj.Name == "" {
		obj.Name = j.Name
	}
	if obj.Namespace == "" {
		obj.Namespace = j.Context.GetNamespace()
	}
	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}
	if _, exists := obj.Annotations["debug"]; !exists {
		obj.Annotations["debug"] = lo.Ternary(j.Debug, "true", "false")
	}
	if _, exists := obj.Annotations["trace"]; !exists {
		obj.Annotations["trace"] = lo.Ternary(j.Trace, "true", "false")
	}

	j.Context = j.Context.WithObject(obj)

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
	cronRunner.Start()
	schedule := j.Schedule
	if override, ok := getProperty(j, j.Context.Properties(), "schedule"); ok {
		schedule = override
	}

	if schedule == "" {
		return fmt.Errorf("job schedule cannot be empty")
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
