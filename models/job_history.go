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
	TimeEnd        time.Time
	Errors         []string `gorm:"-"`
}

func (h *JobHistory) Start() {
	h.TimeStart = time.Now()
	h.Status = StatusRunning
	h.Hostname, _ = os.Hostname()
}

func (h *JobHistory) End() {
	h.DurationMillis = time.Now().Sub(h.TimeStart).Milliseconds()
	h.Details = map[string]any{
		"errors": h.Errors,
	}
	h.Status = StatusFinished
}

func (h *JobHistory) New(name, resourceType, resourceID string) {
	h.Name = name
	h.ResourceType = resourceType
	h.ResourceID = resourceID
}

func (h *JobHistory) AddError(err string) {
	h.ErrorCount += 1
	if err != "" {
		h.Errors = append(h.Errors, err)
	}
}

func (h *JobHistory) IncrSuccess() {
	h.SuccessCount += 1
}
