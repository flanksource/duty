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

var RetentionFew = Retention{
	Success: 1,
	Failed:  3,
}

var RetentionFailed = Retention{
	Success: 0,
	Failed:  3,
}

var RetentionBalanced = Retention{
	Success: 3,
	Failed:  3,
}

var RetentionHigh = Retention{
	Success: 3,
	Failed:  6,
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
	IgnoreSuccessHistory     bool
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
	lock    sync.Mutex
	rings   map[string]*ring.Ring
	evicted chan uuid.UUID

	retention Retention
	singleton bool
}

// populateFromDB syncs the status ring with the existing job histories in db
func (t *StatusRing) populateFromDB(ctx context.Context, name, resourceID string) error {
	var existingHistories []models.JobHistory
	if err := ctx.DB().Where("name = ?", name).Where("resource_id = ?", resourceID).Order("time_start").Find(&existingHistories).Error; err != nil {
		return err
	}
	ctx.Logger.V(4).Infof("found %d histories", len(existingHistories))

	for _, h := range existingHistories {
		t.Add(&h)
	}

	return nil
}

func newStatusRing(r Retention, singleton bool, evicted chan uuid.UUID) StatusRing {
	return StatusRing{
		lock:      sync.Mutex{},
		retention: r,
		rings:     make(map[string]*ring.Ring),
		evicted:   evicted,
		singleton: singleton,
	}
}

func (sr *StatusRing) Add(job *models.JobHistory) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	var r *ring.Ring
	var ok bool
	if r, ok = sr.rings[job.Status]; !ok {
		count := sr.retention.Count(job.Status)
		if sr.singleton && job.Status == models.StatusRunning {
			count = 1
		}

		r = ring.New(count + 1)
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
	// Success is the number of success job history to retain
	Success int

	// Failed is the number of unsuccessful job history to retain
	Failed int

	// Data ...?
	Data bool
}

func (r Retention) Count(status string) int {
	if status == models.StatusSkipped || status == models.StatusFailed || status == models.StatusWarning {
		return r.Failed
	}
	return r.Success
}

func (r Retention) WithData() Retention {
	r.Data = true
	return r
}

func (r Retention) String() string {
	return fmt.Sprintf("success=%d, failed=%d", r.Success, r.Failed)
}

func (r Retention) Empty() bool {
	return r.Success == 0 && r.Failed == 0
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
	j.Context.Counter("job_started", "name", j.Job.Name, "id", j.Job.ResourceID, "resource", j.Job.ResourceType).Add(1)
	j.History = models.NewJobHistory(j.Logger, j.Job.Name, "", "").Start()
	j.Job.LastJob = j.History
	if j.Job.ResourceID != "" {
		j.History.ResourceID = j.Job.ResourceID
	}
	if j.Job.ResourceType != "" {
		j.History.ResourceType = j.Job.ResourceType
	}
	if j.Job.JobHistory && j.Job.Retention.Success > 0 && !j.Job.IgnoreSuccessHistory {
		if err := j.History.Persist(j.FastDB()); err != nil {
			j.Warnf("failed to persist history: %v", err)
		}
	}
}

func (j *JobRuntime) end() {
	j.History.End()
	if j.Job.JobHistory && (j.Job.Retention.Success > 0 || len(j.History.Errors) > 0) && !j.Job.IgnoreSuccessHistory {
		if err := j.History.Persist(j.FastDB()); err != nil {
			j.Warnf("failed to persist history: %v", err)
		}
	}
	j.Job.statusRing.Add(j.History)

	j.Context.Counter("job", "name", j.Job.Name, "id", j.Job.ResourceID, "resource", j.Job.ResourceType, "status", j.History.Status).Add(1)
	j.Context.Histogram("job_duration", context.LongLatencyBuckets, "name", j.Job.Name, "id", j.Job.ResourceID, "resource", j.Job.ResourceType, "status", j.History.Status).Since(j.History.TimeStart)
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
		Retention:  RetentionBalanced,
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

func (j *Job) Run() {
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

	if err := j.init(); err != nil {
		r.Failf("failed to initialize job: %s", r.ID())
		return
	}

	r.start()
	defer r.end()
	if j.Singleton {
		ctx.Logger.V(4).Infof("acquiring lock")

		if j.lock == nil {
			j.lock = &sync.Mutex{}
		}
		if !j.lock.TryLock() {
			r.History.Status = models.StatusSkipped
			ctx.Tracef("failed to acquire lock")
			r.Failf("%s job already running, skipping", r.ID())
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
	if err != nil {
		ctx.Tracef("finished duration=%s, error=%s", time.Since(r.History.TimeStart), err)
		r.History.AddErrorWithSkipReportLevel(err.Error(), 1)
	} else {
		ctx.Tracef("finished duration=%s", time.Since(r.History.TimeStart))
	}
}

func (j *Job) getPropertyNames(key string) []string {
	if j.ID == "" {
		return []string{
			fmt.Sprintf("jobs.%s.%s", j.Name, key),
			fmt.Sprintf("jobs.%s", key)}
	}
	return []string{
		fmt.Sprintf("jobs.%s.%s.%s", j.Name, j.ID, key),
		fmt.Sprintf("jobs.%s.%s", j.Name, key),
		fmt.Sprintf("jobs.%s", key)}
}

func (j *Job) GetProperty(property string) (string, bool) {
	if val := j.Context.Properties().String("jobs."+j.Name+"."+property, ""); val != "" {
		return val, true
	}
	if j.ID != "" {
		if val := j.Context.Properties().String(fmt.Sprintf("jobs.%s.%s.%s", j.Name, j.ID, property), ""); val != "" {
			return val, true
		}
	}
	return "", false
}

func (j *Job) init() error {
	if evictedJobs == nil {
		evictedJobs = make(chan uuid.UUID, 1000)
		go deleteEvictedJobs(j.Context)
	}

	if j.initialized {
		return nil
	}

	j.lastHistoryCleanup = time.Now()

	if schedule, ok := j.GetProperty("schedule"); ok {
		j.Schedule = schedule
	}

	if timeout, ok := j.GetProperty("timeout"); ok {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			j.Context.Warnf("invalid timeout %s", timeout)
		}
		j.Timeout = duration
	}

	j.JobHistory = j.Properties().On(true, j.getPropertyNames("history")...)
	j.Trace = j.Properties().On(false, j.getPropertyNames("trace")...)
	j.Debug = j.Properties().On(false, j.getPropertyNames("debug")...)

	// Set default retention if it is unset
	if j.Retention.Empty() {
		j.Retention = Retention{
			Success: 1,
			Failed:  3,
		}
	}

	if j.ID != "" {
		j.Context = j.Context.WithoutName().WithName(fmt.Sprintf("%s[%s]", j.Name, j.ID))
	} else {
		j.Context = j.Context.WithoutName().WithName(fmt.Sprintf("%s[%s]", j.Name, j.ResourceID))
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

	if dbLevel, ok := j.GetProperty("db-log-level"); ok {
		j.Context = j.Context.WithDBLogLevel(dbLevel)
	}

	j.Context.Tracef("initalized %v", j.String())

	j.statusRing = newStatusRing(j.Retention, j.Singleton, evictedJobs)
	if err := j.statusRing.populateFromDB(j.Context, j.Name, j.ResourceID); err != nil {
		return fmt.Errorf("error populating status ring: %w", err)
	}

	j.initialized = true
	return nil
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

func (j *Job) GetResourcedName() string {
	if j.ID != "" {
		return fmt.Sprintf("%s [%s]", j.Name, j.ID)
	}

	return j.Name
}

func (j *Job) AddToScheduler(cronRunner *cron.Cron) error {
	cronRunner.Start()
	schedule := j.Schedule
	if override, ok := j.GetProperty("schedule"); ok {
		schedule = override
	}

	if schedule == "" {
		return fmt.Errorf("job schedule cannot be empty")
	}

	if schedule == "@never" {
		j.Context.Infof("skipping scheduling")
		return nil
	}

	j.Context.Logger.Named(j.GetResourcedName()).V(1).Infof("scheduled %s", schedule)
	entryID, err := cronRunner.AddJob(schedule, j)
	if err != nil {
		return fmt.Errorf("[%s] failed to schedule job: %s", j.Label(), err)
	}
	j.entryID = &entryID
	if j.RunNow {
		// Run in a goroutine since AddToScheduler should be non-blocking
		defer func() { go j.Run() }()
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
