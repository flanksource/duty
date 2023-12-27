package models

import (
	"fmt"
	"os"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobStatus string

const (
	StatusRunning  = "RUNNING"
	StatusSuccess  = "SUCCESS"
	StatusWarning  = "WARNING"
	StatusFinished = "FINISHED"
	StatusFailed   = "FAILED"
	StatusAborted  = "ABORTED"
)

type JobHistory struct {
	ID             uuid.UUID `gorm:"default:generate_ulid()"`
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
	Errors         []string `gorm:"-"`
}

func (j JobHistory) TableName() string {
	return "job_history"
}

func NewJobHistory(name, resourceType, resourceID string) *JobHistory {
	return &JobHistory{
		Name:         name,
		ResourceType: resourceType,
		ResourceID:   resourceID,
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
	h.Details = map[string]any{
		"errors": h.Errors,
	}

	h.evaluateStatus()
	return h
}

func (h *JobHistory) Persist(db *gorm.DB) error {
	return db.Save(h).Error
}

func (h *JobHistory) AddError(err string) *JobHistory {
	h.ErrorCount += 1
	if err != "" {
		h.Errors = append(h.Errors, err)
	}
	logger.Errorf("%s %s", h, err)
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
	if h.SuccessCount == 0 {
		if h.ErrorCount > 0 {
			h.Status = StatusFailed
		} else {
			h.Status = StatusFinished
		}
	} else {
		if h.ErrorCount == 0 {
			h.Status = StatusSuccess
		} else {
			h.Status = StatusWarning
		}
	}
}
