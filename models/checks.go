package models

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Check struct {
	ID          uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	CanaryID    uuid.UUID           `json:"canary_id"`
	Spec        types.JSON          `json:"-"`
	Type        string              `json:"type"`
	Name        string              `json:"name"`
	CanaryName  string              `json:"canary_name" gorm:"-"`
	Namespace   string              `json:"namespace"  gorm:"-"`
	Labels      types.JSONStringMap `json:"labels" gorm:"type:jsonstringmap"`
	Description string              `json:"description,omitempty"`
	Status      string              `json:"status,omitempty"`
	Uptime      Uptime              `json:"uptime"  gorm:"-"`
	Latency     Latency             `json:"latency"  gorm:"-"`
	Statuses    []CheckStatus       `json:"checkStatuses"  gorm:"-"`
	Owner       string              `json:"owner,omitempty"`
	Severity    string              `json:"severity,omitempty"`
	Icon        string              `json:"icon,omitempty"`
	DisplayType string              `json:"displayType,omitempty"  gorm:"-"`
	LastRuntime *time.Time          `json:"lastRuntime,omitempty"`
	NextRuntime *time.Time          `json:"nextRuntime,omitempty"`
	UpdatedAt   *time.Time          `json:"updatedAt,omitempty"`
	CreatedAt   *time.Time          `json:"createdAt,omitempty"`
	DeletedAt   *time.Time          `json:"deletedAt,omitempty"`
}

func (c Check) ToString() string {
	return fmt.Sprintf("%s-%s-%s", c.Name, c.Type, c.Description)
}

func (c Check) GetDescription() string {
	return c.Description
}

type Checks []*Check

func (c Checks) Len() int {
	return len(c)
}

func (c Checks) Less(i, j int) bool {
	return c[i].ToString() < c[j].ToString()
}

func (c Checks) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Checks) Find(key string) *Check {
	for _, check := range c {
		if check.Name == key {
			return check
		}
	}
	return nil
}

type Uptime struct {
	Passed int     `json:"passed"`
	Failed int     `json:"failed"`
	P100   float64 `json:"p100,omitempty"`
}

func (u Uptime) String() string {
	if u.Passed == 0 && u.Failed == 0 {
		return ""
	}
	if u.Passed == 0 {
		return fmt.Sprintf("0/%d 0%%", u.Failed)
	}
	percentage := 100.0 * (1 - (float64(u.Failed) / float64(u.Passed+u.Failed)))
	return fmt.Sprintf("%d/%d (%0.1f%%)", u.Passed, u.Passed+u.Failed, percentage)
}
