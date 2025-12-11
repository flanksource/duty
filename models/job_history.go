package models

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type JobStatus string

const (
	StatusRunning = "RUNNING"
	StatusSuccess = "SUCCESS"
	StatusWarning = "WARNING"
	StatusFailed  = "FAILED"
	StatusStale   = "STALE"
	StatusSkipped = "SKIPPED"
)

func (j JobStatus) Pretty() api.Text {
	var icon, style string
	switch j {
	case StatusSuccess:
		icon, style = "✓", "text-green-600 font-bold"
	case StatusFailed:
		icon, style = "✗", "text-red-600 font-bold"
	case StatusWarning:
		icon, style = "!", "text-yellow-600 font-bold"
	case StatusRunning:
		icon, style = "▶", "text-blue-600"
	case StatusStale:
		icon, style = "⏱", "text-gray-500"
	case StatusSkipped:
		icon, style = "⊘", "text-gray-400"
	default:
		icon, style = "•", "text-gray-500"
	}
	return clicky.Text(icon+" ", style).Append(string(j), style)
}

type JobHistory struct {
	ID             uuid.UUID `gorm:"default:generate_ulid()"`
	AgentID        uuid.UUID `json:"agent_id,omitempty"`
	Name           string
	SuccessCount   int
	ErrorCount     int
	Hostname       string
	DurationMillis int64
	ResourceType   string
	ResourceID     string
	Details        types.JSONMap
	Status         string
	TimeStart      time.Time
	TimeEnd        *time.Time
	Errors         []string      `gorm:"-"`
	Logger         logger.Logger `gorm:"-"`
}

func (j JobHistory) PK() string {
	return j.ID.String()
}

func (j JobHistory) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []JobHistory
	err := db.Where("is_pushed IS FALSE").Where("status IN (?,?,?)", StatusFailed, StatusWarning, StatusSuccess).Find(&items).Error
	return lo.Map(items, func(i JobHistory, _ int) DBTable { return i }), err
}

func (j JobHistory) AsError() error {
	if len(j.Errors) == 0 {
		return nil
	}
	return errors.New(strings.Join(j.Errors, ","))
}

func (j JobHistory) TableName() string {
	return "job_history"
}

func (j JobHistory) GetAgentID() string {
	if j.AgentID == uuid.Nil {
		return ""
	}
	return j.AgentID.String()
}

func NewJobHistory(log logger.Logger, name, resourceType, resourceID string) *JobHistory {
	return &JobHistory{
		Name:         name,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Logger:       log,
	}
}

func (h *JobHistory) Start() *JobHistory {
	h.TimeStart = time.Now()
	h.Status = StatusRunning
	h.Hostname, _ = os.Hostname()
	return h
}

func (h *JobHistory) End() *JobHistory {
	timeEnd := time.Now()
	h.TimeEnd = &timeEnd
	h.DurationMillis = timeEnd.Sub(h.TimeStart).Milliseconds()

	if len(h.Errors) > 0 {
		if h.Details == nil {
			h.Details = make(map[string]any)
		}
		h.Details["errors"] = h.Errors
	}

	h.evaluateStatus()
	return h
}

func (h *JobHistory) Persist(db *gorm.DB) error {
	return db.Save(h).Error
}

func (h *JobHistory) AddDetails(key string, val any) {
	if h.Details == nil {
		h.Details = make(map[string]any)
	}

	h.Details[key] = val
}

func (h *JobHistory) AddErrorf(msg string, args ...interface{}) *JobHistory {
	err := fmt.Sprintf(msg, args...)
	h.ErrorCount += 1
	if err != "" {
		h.Errors = append(h.Errors, err)
	}
	h.Logger.WithSkipReportLevel(1).Errorf("%s %s", h, err)
	return h
}

func (h *JobHistory) AddError(err any) *JobHistory {
	h.ErrorCount += 1
	switch v := err.(type) {
	case error:
		h.Errors = append(h.Errors, v.Error())
	case string:
		h.Errors = append(h.Errors, v)
	default:
	}
	h.Logger.Errorf("%s %v", h, err)
	return h
}

func (h *JobHistory) AddErrorWithSkipReportLevel(err string, level int) *JobHistory {
	h.ErrorCount += 1
	if err != "" {
		h.Errors = append(h.Errors, err)
	}
	h.Logger.WithSkipReportLevel(level).Errorf("%s %s", h, err)
	return h
}

func (h *JobHistory) String() string {
	if h.ResourceID != "" {
		return fmt.Sprintf("%s{%s}", h.Name, h.End().ResourceID)
	}
	return h.Name
}

func (h *JobHistory) IncrSuccess() *JobHistory {
	h.SuccessCount += 1
	return h
}

// EvaluateStatus updates the Status field of JobHistory based on the counts of
// Success and Error in it.
func (h *JobHistory) evaluateStatus() {
	if h.Status != StatusRunning {
		return
	}

	if h.ErrorCount > 0 && h.SuccessCount > 0 {
		h.Status = StatusWarning
	} else if h.ErrorCount > 0 && h.SuccessCount == 0 {
		h.Status = StatusFailed
	} else {
		h.Status = StatusSuccess
	}
}
