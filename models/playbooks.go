package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

var AllowedColumnFieldsInPlaybooks = []string{"category"}

// PlaybookRunStatus are statuses for a playbook run and its actions.
type PlaybookRunStatus string

const (
	PlaybookRunStatusCancelled       PlaybookRunStatus = "cancelled"
	PlaybookRunStatusTimedOut        PlaybookRunStatus = "timed_out"
	PlaybookRunStatusCompleted       PlaybookRunStatus = "completed"
	PlaybookRunStatusFailed          PlaybookRunStatus = "failed"
	PlaybookRunStatusPendingApproval PlaybookRunStatus = "pending_approval"
	PlaybookRunStatusRunning         PlaybookRunStatus = "running"
	PlaybookRunStatusScheduled       PlaybookRunStatus = "scheduled"
	PlaybookRunStatusSleeping        PlaybookRunStatus = "sleeping"
	PlaybookRunStatusRetrying        PlaybookRunStatus = "retrying"
	PlaybookRunStatusWaiting         PlaybookRunStatus = "waiting" // waiting for a consumer
)

// PlaybookRunStatus are statuses for a playbook run and its actions.
type PlaybookActionStatus string

const (
	// Waiting for child playbook runs to complete
	PlaybookActionStatusWaitingChildren PlaybookActionStatus = "waiting_children"
	PlaybookActionStatusCompleted       PlaybookActionStatus = "completed"
	PlaybookActionStatusFailed          PlaybookActionStatus = "failed"
	PlaybookActionStatusRunning         PlaybookActionStatus = "running"
	PlaybookActionStatusScheduled       PlaybookActionStatus = "scheduled"
	PlaybookActionStatusWaiting         PlaybookActionStatus = "waiting" // Waiting for agents
	PlaybookActionStatusSkipped         PlaybookActionStatus = "skipped"
	PlaybookActionStatusSleeping        PlaybookActionStatus = "sleeping"
)

var PlaybookActionFinalStates = []PlaybookActionStatus{
	PlaybookActionStatusFailed,
	PlaybookActionStatusCompleted,
	PlaybookActionStatusSkipped,
}

func (p Playbook) TableName() string {
	return "playbooks"
}

func (p Playbook) PK() string {
	return p.ID.String()
}

var PlaybookRunStatusFinalStates = []PlaybookRunStatus{
	PlaybookRunStatusCancelled,
	PlaybookRunStatusCompleted,
	PlaybookRunStatusFailed,
	PlaybookRunStatusTimedOut,
}

var PlaybookRunStatusExecutingGroup = []PlaybookRunStatus{
	PlaybookRunStatusRunning,
	PlaybookRunStatusScheduled,
	PlaybookRunStatusSleeping,
	PlaybookRunStatusRetrying,
	PlaybookRunStatusWaiting,
	PlaybookRunStatusPendingApproval,
}

var _ types.ResourceSelectable = &Playbook{}

type Playbook struct {
	ID          uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	Namespace   string              `json:"namespace"`
	Name        string              `json:"name"`
	Title       string              `json:"title"`
	Icon        string              `json:"icon,omitempty"`
	Description string              `json:"description,omitempty"`
	Spec        types.JSON          `json:"spec"`
	Source      string              `json:"source"`
	Category    string              `json:"category"`
	Tags        types.JSONStringMap `json:"tags,omitempty" gorm:"default:NULL"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt   time.Time           `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt   time.Time           `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (p Playbook) SelectableFields() map[string]any {
	return map[string]any{
		"category": p.Category,
	}
}

func (p *Playbook) GetFieldsMatcher() fields.Fields {
	return types.GenericFieldMatcher{Fields: p.SelectableFields()}
}

func (p *Playbook) GetLabelsMatcher() labels.Labels {
	return noopMatcher{}
}

func (p *Playbook) GetTagsMatcher() labels.Labels {
	return types.GenericLabelsMatcher{Map: p.Tags}
}

func (p *Playbook) GetName() string {
	return p.Name
}

func (p *Playbook) GetNamespace() string {
	return p.Namespace
}

func (p *Playbook) GetID() string {
	return p.ID.String()
}

func (p *Playbook) GetType() string {
	return ""
}

func (p *Playbook) GetStatus() (string, error) {
	return "", nil
}

func (p *Playbook) GetHealth() (string, error) {
	return string(HealthUnknown), nil
}

func (p *Playbook) NamespacedName() string {
	if p.Namespace != "" {
		return fmt.Sprintf("%s/%s", p.Namespace, p.Name)
	}

	return p.Name
}

func (p *Playbook) LoggerName() string {
	return "playbook." + p.Name
}

func (p Playbook) Context() map[string]any {
	return map[string]any{
		"playbook_id": p.ID.String(),
		"namespace":   p.Namespace,
		"name":        p.Name,
	}
}

func (p *Playbook) Save(db *gorm.DB) error {
	if p.ID != uuid.Nil {
		return db.Model(p).Clauses(
			clause.Returning{},
		).Save(p).Error
	}
	return db.Model(p).Clauses(
		clause.Returning{},
		clause.OnConflict{
			Columns:     []clause.Column{{Name: "namespace"}, {Name: "name"}, {Name: "category"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
			UpdateAll:   true,
		}).Create(p).Error
}

func (p Playbook) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookRun struct {
	ID            uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	PlaybookID    uuid.UUID           `json:"playbook_id"`
	Status        PlaybookRunStatus   `json:"status,omitempty"`
	Spec          types.JSON          `json:"spec"`
	CreatedAt     time.Time           `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	StartTime     *time.Time          `json:"start_time,omitempty" time_format:"postgres_timestamp"`
	ScheduledTime time.Time           `json:"scheduled_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW(), NOT NULL"`
	EndTime       *time.Time          `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	Timeout       time.Duration       `json:"timeout,omitempty"`
	CreatedBy     *uuid.UUID          `json:"created_by,omitempty"`
	ComponentID   *uuid.UUID          `json:"component_id,omitempty"`
	CheckID       *uuid.UUID          `json:"check_id,omitempty"`
	ConfigID      *uuid.UUID          `json:"config_id,omitempty"`
	Error         *string             `json:"error,omitempty"`
	Parameters    types.JSONStringMap `json:"parameters,omitempty" gorm:"default:null"`
	Request       types.JSONMap       `json:"request,omitempty" gorm:"default:null"`
	AgentID       *uuid.UUID          `json:"agent_id,omitempty"`

	// Parent Run's id
	ParentID *uuid.UUID `json:"parent_id,omitempty"`

	// Parent notification send's id
	NotificationSendID *uuid.UUID `json:"notification_send_id,omitempty"`
}

func (p PlaybookRun) TableName() string {
	return "playbook_runs"
}

func (p PlaybookRun) PK() string {
	return p.ID.String()
}

func (p PlaybookRun) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

func (p PlaybookRun) Update(db *gorm.DB, columns map[string]any) error {
	return oops.Tags("db").Wrap(db.Model(PlaybookRun{}).Where("id = ?", p.ID).UpdateColumns(columns).Error)
}

func (p PlaybookRun) Schedule(db *gorm.DB) error {
	return p.Update(db, map[string]any{
		"status":         PlaybookRunStatusScheduled,
		"scheduled_time": gorm.Expr("CLOCK_TIMESTAMP()"),
	})
}

func (p PlaybookRun) Retry(db *gorm.DB, delay time.Duration) error {
	return p.Update(db, map[string]any{
		"status":         PlaybookRunStatusRetrying,
		"start_time":     gorm.Expr("CASE WHEN start_time IS NULL THEN CLOCK_TIMESTAMP() ELSE start_time END"),
		"scheduled_time": gorm.Expr(fmt.Sprintf("CLOCK_TIMESTAMP() + INTERVAL '%d SECONDS'", int(delay.Seconds()))),
	})
}

func (p PlaybookRun) Delay(db *gorm.DB, delay time.Duration) error {
	return p.Update(db, map[string]any{
		"status":         PlaybookRunStatusSleeping,
		"start_time":     gorm.Expr("CASE WHEN start_time IS NULL THEN CLOCK_TIMESTAMP() ELSE start_time END"),
		"scheduled_time": gorm.Expr(fmt.Sprintf("CLOCK_TIMESTAMP() + INTERVAL '%d SECONDS'", int(delay.Seconds()))),
	})
}

func (p PlaybookRun) Waiting(db *gorm.DB) error {
	return p.Update(db, map[string]any{
		"status":     PlaybookRunStatusWaiting,
		"start_time": gorm.Expr("CASE WHEN start_time IS NULL THEN CLOCK_TIMESTAMP() ELSE start_time END"),
	})
}

func (p PlaybookRun) Running(db *gorm.DB) error {
	return p.Update(db, map[string]any{
		"status":     PlaybookRunStatusRunning,
		"start_time": gorm.Expr("CASE WHEN start_time IS NULL THEN CLOCK_TIMESTAMP() ELSE start_time END"),
	})
}

func (p PlaybookRun) End(db *gorm.DB) error {
	return p.endWithDeterminedStatus(db)
}

func (p PlaybookRun) Cancel(db *gorm.DB) error {
	return p.endWithStatus(db, PlaybookRunStatusCancelled)
}

// endWithDeterminedStatus ends the playbook run with a status determined by its actions
func (p PlaybookRun) endWithDeterminedStatus(db *gorm.DB) error {
	status := PlaybookRunStatusCompleted
	var statuses []PlaybookActionStatus
	if err := db.Select("status").Model(&PlaybookRunAction{}).Where("playbook_run_id = ?", p.ID).Find(&statuses).Error; err != nil {
		return oops.Tags("db").Wrap(err)
	}

	if _, failed := lo.Find(statuses, func(i PlaybookActionStatus) bool { return i == PlaybookActionStatusFailed }); failed {
		status = PlaybookRunStatusFailed
	}

	return p.endWithStatus(db, status)
}

// endWithStatus ends the playbook run with the specified status
func (p PlaybookRun) endWithStatus(db *gorm.DB, status PlaybookRunStatus) error {
	if err := p.Update(db, map[string]any{
		"status":   status,
		"end_time": gorm.Expr("CLOCK_TIMESTAMP()"),
	}); err != nil {
		return err
	}

	if p.NotificationSendID != nil {
		updates := map[string]any{}
		if status == PlaybookRunStatusFailed {
			updates["status"] = NotificationStatusError
			updates["error"] = "playbook failed"
		} else {
			updates["status"] = NotificationStatusSent
		}

		if err := db.Model(&NotificationSendHistory{}).Where("id = ?", *p.NotificationSendID).Updates(updates).Error; err != nil {
			return err
		}

		var notif Notification
		var sendHistory NotificationSendHistory
		if err := db.Where("id = ?", *p.NotificationSendID).First(&sendHistory).Error; err != nil {
			return fmt.Errorf("failed to get notification send history: %w", err)
		}
		if err := db.Where("id = ?", sendHistory.NotificationID).First(&notif).Error; err != nil {
			return fmt.Errorf("failed to get notification: %w", err)
		}

		if status == PlaybookRunStatusFailed && notif.HasFallbackSet() {
			if err := GenerateFallbackAttempt(db, notif, sendHistory); err != nil {
				return fmt.Errorf("failed to generate fallback attempt: %w", err)
			}
		}
	}

	if p.ParentID != nil {
		parentRun := PlaybookRun{ID: *p.ParentID}
		if err := parentRun.resumeChildrenWaitingAction(db); err != nil {
			return fmt.Errorf("failed to resume action awaiting children: %w", err)
		}
	}

	return nil
}

// resumeChildrenWaitingAction resumes the action that's awaiting children
// if all its children have terminated.
func (p PlaybookRun) resumeChildrenWaitingAction(db *gorm.DB) error {
	query := `
	SELECT COUNT(*)
	FROM playbook_runs AS parent
	WHERE parent.id = ?
	AND parent.status = ?
	AND NOT EXISTS (
		SELECT 1
		FROM playbook_runs AS child
		WHERE child.parent_id = parent.id
		AND child.status NOT IN (?)
	)
	`

	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Raw(query, p.ID, PlaybookRunStatusRunning, PlaybookRunStatusFinalStates).Scan(&count).Error; err != nil {
			return fmt.Errorf("failed to query parent playbook runs: %w", err)
		}

		if count == 0 {
			return nil
		}

		// Reschedule the action that's awaiting children
		if err := tx.Model(&PlaybookRunAction{}).
			Where("playbook_run_id = ?", p.ID).
			Where("status = ?", PlaybookActionStatusWaitingChildren).
			Update("status", PlaybookActionStatusScheduled).Error; err != nil {
			return fmt.Errorf("failed to update parent playbook runs: %w", err)
		}

		return nil
	})
}

func (p PlaybookRun) Assign(db *gorm.DB, agent *Agent, action string) error {
	runAction := PlaybookRunAction{
		PlaybookRunID: p.ID,
		Name:          action,
		Status:        PlaybookActionStatusWaiting,
		AgentID:       &agent.ID,
	}
	if err := db.Save(&runAction).Error; err != nil {
		return err
	}
	return p.Waiting(db)
}

func (p PlaybookRun) RetryAction(db *gorm.DB, action string, retryCount int) (*PlaybookRunAction, error) {
	runAction := PlaybookRunAction{
		PlaybookRunID: p.ID,
		Name:          action,
		Status:        PlaybookActionStatusScheduled,
		RetryCount:    retryCount,
	}
	if err := db.Save(&runAction).Error; err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}
	return &runAction, p.Running(db)
}

func (p PlaybookRun) StartAction(db *gorm.DB, action string) (*PlaybookRunAction, error) {
	runAction := PlaybookRunAction{
		PlaybookRunID: p.ID,
		Name:          action,
		Status:        PlaybookActionStatusScheduled,
	}
	if err := db.Save(&runAction).Error; err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}
	return &runAction, p.Running(db)
}

func (p PlaybookRun) Fail(db *gorm.DB, err error) error {
	return p.Update(db, map[string]any{
		"error":    err.Error(),
		"end_time": gorm.Expr("CLOCK_TIMESTAMP()"),
		"status":   PlaybookRunStatusFailed,
	})
}

func (p PlaybookRun) GetActions(db *gorm.DB) (actions []PlaybookRunAction, err error) {
	err = db.Model(actions).Where("playbook_run_id = ?", p.ID).Find(&actions).Error
	if err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}

	return actions, err
}

func (p PlaybookRun) GetAgentActions(db *gorm.DB) (actions []PlaybookRunAction, err error) {
	err = db.Raw(`
		SELECT * FROM playbook_run_actions
		INNER JOIN playbook_action_agent_data ON
			playbook_run_actions.id = playbook_action_agent_data.action_id
		WHERE playbook_action_agent_data.run_id = ?`, p.ID).Scan(&actions).Error
	if err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}

	return actions, err
}

func (p PlaybookRun) GetAgentAction(db *gorm.DB, name string) (*PlaybookRunAction, error) {
	actions, err := p.GetAgentActions(db)
	if err != nil {
		return nil, err
	}
	for _, action := range actions {
		if action.Name == name {
			return &action, nil
		}
	}
	return nil, oops.Errorf("action not found: %s, available actions [%s]", name, strings.Join(
		lo.Map(actions, func(i PlaybookRunAction, _ int) string { return i.Name }), ", "))
}

func (p PlaybookRunAction) Load(db *gorm.DB) (*PlaybookRunAction, error) {
	var _refreshed []PlaybookRunAction
	err := db.Model(p).Where("id = ?", p.ID).Find(&_refreshed).Error
	if err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}
	if len(_refreshed) > 0 {
		return &_refreshed[0], err
	}
	return nil, oops.Tags("db").Errorf("Playbook run action '%v' not found", p.ID)
}

func (p Playbook) Load(db *gorm.DB) (*Playbook, error) {
	var _refreshed []Playbook
	err := db.Model(p).Where("id = ?", p.ID).Find(&_refreshed).Error
	if err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}
	if len(_refreshed) > 0 {
		return &_refreshed[0], err
	}
	return nil, oops.Tags("db").Errorf("Playbook run action '%v' not found", p.ID)
}

func (p PlaybookRun) Load(db *gorm.DB) (*PlaybookRun, error) {
	var _refreshed []PlaybookRun
	err := db.Model(p).Where("id = ?", p.ID).Find(&_refreshed).Error
	if err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}
	if len(_refreshed) > 0 {
		return &_refreshed[0], err
	}
	return nil, oops.Tags("db").Errorf("Playbook run '%v' not found", p.ID)
}

func (p PlaybookRun) GetAction(db *gorm.DB, name string) (action *PlaybookRunAction, err error) {
	actions, err := p.GetActions(db)
	if err != nil {
		return nil, err
	}
	for _, action := range actions {
		if action.Name == name {
			return &action, nil
		}
	}
	return nil, oops.Errorf("action not found: %s, available actions [%s]", name, strings.Join(
		lo.Map(actions, func(i PlaybookRunAction, _ int) string { return i.Name }), ", "))
}

func (p PlaybookRun) Context() map[string]any {
	return map[string]any{
		"run_id":      p.ID.String(),
		"playbook_id": p.PlaybookID,
	}
}

func (p *PlaybookRun) String(db *gorm.DB) string {
	var s string
	playbook, _ := p.GetPlaybook(db)
	if playbook != nil {
		s += fmt.Sprintf("%s %s id=%s\n", playbook.Name, colorStatus(string(p.Status)), p.ID)
	} else {
		s += fmt.Sprintf("Playbook %s id=%s\n", colorStatus(string(p.Status)), p.ID)
	}

	actions, _ := p.GetActions(db)
	for _, action := range actions {
		s += fmt.Sprintf("\t\t%s\n", &action)
	}
	return s
}

func (run *PlaybookRun) GetABACAttributes(db *gorm.DB) (*ABACAttribute, error) {
	var output ABACAttribute

	var playbook Playbook
	if err := db.First(&playbook, run.PlaybookID).Error; err != nil {
		return nil, err
	}
	output.Playbook = playbook

	if run.ComponentID != nil {
		var component Component
		if err := db.First(&component, run.ComponentID).Error; err != nil {
			return nil, err
		}
		output.Component = component
	}

	if run.CheckID != nil {
		var check Check
		if err := db.First(&check, run.CheckID).Error; err != nil {
			return nil, err
		}
		output.Check = check
	}

	if run.ConfigID != nil {
		var config ConfigItem
		if err := db.First(&config, run.ConfigID).Error; err != nil {
			return nil, err
		}
		output.Config = config
	}

	return &output, nil
}

type PlaybookRunAction struct {
	ID            uuid.UUID            `json:"id" gorm:"default:generate_ulid()"`
	Name          string               `json:"name" gorm:"not null"`
	PlaybookRunID uuid.UUID            `json:"playbook_run_id"`
	Status        PlaybookActionStatus `json:"status,omitempty"`
	ScheduledTime time.Time            `json:"scheduled_time,omitempty" time_format:"postgres_timestamp" gorm:"default:NOW(), NOT NULL"`
	StartTime     time.Time            `json:"start_time,omitempty" time_format:"postgres_timestamp"  gorm:"default:NOW(), NOT NULL"`
	EndTime       *time.Time           `json:"end_time,omitempty" time_format:"postgres_timestamp"`
	Result        types.JSONMap        `json:"result,omitempty" gorm:"default:null"`
	Error         *string              `json:"error,omitempty" gorm:"default:null"`
	IsPushed      bool                 `json:"is_pushed"`
	AgentID       *uuid.UUID           `json:"agent_id,omitempty"`

	// RetryCount represents the Nth retry of this action
	RetryCount int `json:"attempt,omitempty" gorm:"default:NULL"`
}

func (p PlaybookRunAction) JSON() (out map[string]any) {
	if stdout, ok := p.Result["stdout"]; ok {
		_ = json.Unmarshal([]byte(stdout.(string)), &out)
	}
	return out
}

func (p PlaybookRunAction) String() string {
	return fmt.Sprintf("%s %s %s", p.Name, colorStatus(string(p.Status)), lo.FromPtrOr(p.Error, ""))
}

func colorStatus(s string) string {
	switch s {
	case string(PlaybookActionStatusScheduled):
		return "scheduled"
	case string(PlaybookActionStatusWaiting):
		return console.BrightYellowf("waiting")
	case string(PlaybookActionStatusRunning):
		return console.BrightGreenf("running")
	case string(PlaybookActionStatusCompleted):
		return console.BrightGreenf("completed")
	case string(PlaybookActionStatusFailed):
		return console.Redf("failed")
	}
	return s
}

func (p *Playbook) NamespaceScope() string {
	return p.Namespace
}

func (p PlaybookRunAction) Context() map[string]any {
	return map[string]any{
		"action_id":   p.ID.String(),
		"action_name": p.Name,
		"run_id":      p.PlaybookRunID.String(),
	}
}

func (p PlaybookRun) GetPlaybook(db *gorm.DB) (*Playbook, error) {
	var playbook Playbook
	err := db.Model(playbook).Where("id = ?", p.PlaybookID).First(&playbook).Error
	return &playbook, oops.Tags("db").Wrap(err)
}

func (p PlaybookRunAction) GetPlaybook(db *gorm.DB) (*Playbook, error) {
	var playbook Playbook
	err := db.Table("playbook_runs").
		Select("playbooks.*").
		Joins("LEFT JOIN playbooks ON playbooks.id = playbook_runs.playbook_id").
		Where("playbook_runs.id = ?", p.PlaybookRunID).
		First(&playbook).Error
	return &playbook, oops.Tags("db").Wrap(err)
}

func (p PlaybookRunAction) GetRun(db *gorm.DB) (*PlaybookRun, error) {
	var run PlaybookRun
	err := db.Where("id = ?", p.PlaybookRunID).First(&run).Error
	return &run, oops.Tags("db").Wrap(err)
}

func (p PlaybookRunAction) Start(db *gorm.DB) error {
	return p.Update(db, map[string]any{
		"start_time": gorm.Expr("CASE WHEN start_time IS NULL THEN CLOCK_TIMESTAMP() ELSE start_time END"),
		"status":     PlaybookActionStatusRunning,
	})
}

func (p PlaybookRunAction) Fail(db *gorm.DB, result any, err error) error {
	updates := map[string]any{
		"result":     marshallResult(result),
		"start_time": gorm.Expr("CASE WHEN start_time IS NULL THEN CLOCK_TIMESTAMP() ELSE start_time END"),
		"end_time":   gorm.Expr("CLOCK_TIMESTAMP()"),
		"status":     PlaybookActionStatusFailed,
	}

	if err != nil {
		if o, ok := oops.AsOops(err); ok {
			updates["result"] = map[string]any{
				"result": result,
				"error":  o.ToMap(),
			}
		}
		updates["error"] = err.Error()
	}

	if err := p.Update(db, updates); err != nil {
		return err
	}

	return p.ScheduleRun(db)
}

func (p PlaybookRunAction) Skip(db *gorm.DB) error {
	if err := p.Update(db, map[string]any{
		"end_time": gorm.Expr("CLOCK_TIMESTAMP()"),
		"status":   PlaybookActionStatusSkipped,
	}); err != nil {
		return nil
	}

	return p.ScheduleRun(db)
}

func marshallResult(result any) string {
	if result == nil || result == "" {
		return "{}"
	}

	var maybeJson string
	switch v := result.(type) {
	case string:
		maybeJson = v
	case []byte:
		if len(v) == 0 {
			return "{}"
		}
		maybeJson = string(v)
	default:
		b, _ := json.Marshal(result)
		return string(b)
	}
	var to any
	if err := json.Unmarshal([]byte(maybeJson), &to); err == nil {
		return maybeJson
	}
	b, _ := json.Marshal(map[string]any{
		"result": maybeJson,
	})
	return string(b)
}

func (p PlaybookRunAction) WaitForChildren(db *gorm.DB) error {
	return p.Update(db, map[string]any{
		"status": PlaybookActionStatusWaitingChildren,
	})
}

func (p PlaybookRunAction) Complete(db *gorm.DB, result any) error {
	if err := p.Update(db, map[string]any{
		"result":   marshallResult(result),
		"end_time": gorm.Expr("CLOCK_TIMESTAMP()"),
		"status":   PlaybookActionStatusCompleted,
	}); err != nil {
		return err
	}

	return p.ScheduleRun(db)
}

func (p PlaybookRunAction) Update(db *gorm.DB, columns map[string]any) error {
	return oops.Tags("db").Wrap(db.Model(PlaybookRunAction{}).Where("id = ?", p.ID).UpdateColumns(columns).Error)
}

func (p PlaybookRunAction) UpdateRun(db *gorm.DB, columns map[string]any) error {
	return PlaybookRun{ID: p.PlaybookRunID}.Update(db, columns)
}

func (p PlaybookRunAction) ScheduleRun(db *gorm.DB) error {
	return PlaybookRun{ID: p.PlaybookRunID}.Schedule(db)
}

func (p PlaybookRunAction) TableName() string {
	return "playbook_run_actions"
}

func (p PlaybookRunAction) PK() string {
	return p.ID.String()
}

func (p PlaybookRunAction) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookApproval struct {
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	RunID     uuid.UUID  `json:"run_id"`
	PersonID  *uuid.UUID `json:"person_id,omitempty"`
	TeamID    *uuid.UUID `json:"team_id,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:false"`
}

func (p PlaybookApproval) TableName() string {
	return "playbook_approvals"
}

func (p PlaybookApproval) PK() string {
	return p.ID.String()
}

func (p PlaybookApproval) AsMap(removeFields ...string) map[string]any {
	return asMap(p, removeFields...)
}

type PlaybookActionAgentData struct {
	ActionID   uuid.UUID  `json:"action_id"`
	RunID      uuid.UUID  `json:"run_id"`
	PlaybookID uuid.UUID  `json:"playbook_id"`
	Spec       types.JSON `json:"spec"`
	Env        types.JSON `json:"env,omitempty"`
}

func (p *PlaybookActionAgentData) Context() map[string]any {
	if p == nil {
		return nil
	}
	return map[string]any{
		"action_id":   p.ActionID.String(),
		"run_id":      p.RunID.String(),
		"playbook_id": p.PlaybookID.String(),
	}
}

func (t *PlaybookActionAgentData) TableName() string {
	return "playbook_action_agent_data"
}
