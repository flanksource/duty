package query

import "github.com/flanksource/duty/models"

const DefaultDepth = 5

type TopologyQuerySortBy string

const (
	TopologyQuerySortByName  TopologyQuerySortBy = "name"
	TopologyQuerySortByField TopologyQuerySortBy = "field:"
)

type TopologyOptions struct {
	ID      string
	Owner   string
	Labels  map[string]string
	AgentID string
	Flatten bool
	Depth   int
	// TODO: Filter status and types in DB Query
	Types  []string
	Status []string

	SortBy    TopologyQuerySortBy
	SortOrder string

	// when set to true, only the children (except the direct children) are returned.
	// when set to false, the direct children & the parent itself is fetched.
	nonDirectChildrenOnly bool
}

// Map of tag keys to the list of available values
type Tags map[string][]string

// +kubebuilder:object:generate=true
type TopologyResponse struct {
	Components     models.Components `json:"components"`
	HealthStatuses []string          `json:"healthStatuses"`
	Teams          []string          `json:"teams"`
	Tags           Tags              `json:"tags"`
	Types          []string          `json:"types"`
}
