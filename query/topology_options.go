package query

import (
	"encoding/json"

	"github.com/flanksource/commons/hash"
	"github.com/flanksource/duty/models"
)

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
	NoCache   bool

	// when set to true, only the children (except the direct children) are returned.
	// when set to false, the direct children & the parent itself is fetched.
	nonDirectChildrenOnly bool
}

func (opt TopologyOptions) CacheKey() string {
	// these options are applied post db query
	opt.SortBy = ""
	opt.SortOrder = ""
	opt.Depth = 0
	opt.Flatten = false
	opt.NoCache = false
	opt.Types = []string{}
	opt.Status = []string{}
	data, _ := json.Marshal(opt)
	return hash.Sha256Hex(string(data))
}

// Map of tag keys to the list of available values
type Tags map[string][]string

type TopologyResponse struct {
	Components     models.Components `json:"components"`
	HealthStatuses []string          `json:"healthStatuses"`
	Teams          []string          `json:"teams"`
	Tags           Tags              `json:"tags"`
	Types          []string          `json:"types"`
}
