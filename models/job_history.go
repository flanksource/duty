package models

import (
	"os"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

const (
	StatusRunning  = "RUNNING"
	StatusFinished = "FINISHED"
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
	h.Status = StatusFinished
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
