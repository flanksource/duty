package models

import (
	"os"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

const (
	StatusRunning  = "RUNNING"
	StatusSuccess  = "SUCCESS"
	StatusWarning  = "WARNING"
	StatusFinished = "FINISHED"
	StatusFailed   = "FAILED"
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

	// Set success count if not set before
	if h.SuccessCount == 0 && h.ErrorCount == 0 {
		h.IncrSuccess()
	}

	h.evaluateStatus()
	return h
}

func (h *JobHistory) AddError(err string) *JobHistory {
	h.ErrorCount += 1
	if err != "" {
		h.Errors = append(h.Errors, err)
	}
	return h
}

func (h *JobHistory) IncrSuccess() *JobHistory {
	h.SuccessCount += 1
	return h
}

// EvaluateStatus updates the Status field of JobHistory based on the counts of
// Success and Error in it.
func (h *JobHistory) evaluateStatus() {
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
