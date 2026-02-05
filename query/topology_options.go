package query

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/flanksource/commons/collections"
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

func NewTopologyParams(values url.Values) TopologyOptions {
	parseItems := func(items string) []string {
		if strings.TrimSpace(items) == "" {
			return nil
		}
		return strings.Split(strings.TrimSpace(items), ",")
	}

	var labels map[string]string
	if values.Get("labels") != "" {
		labels = collections.KeyValueSliceToMap(strings.Split(values.Get("labels"), ","))
	}

	var err error
	var depth = DefaultDepth
	if depthStr := values.Get("depth"); depthStr != "" {
		depth, err = strconv.Atoi(depthStr)
		if err != nil {
			depth = DefaultDepth
		}
	}
	return TopologyOptions{
		ID:        values.Get("id"),
		Owner:     values.Get("owner"),
		Labels:    labels,
		Status:    parseItems(values.Get("status")),
		Depth:     depth,
		Types:     parseItems(values.Get("type")),
		Flatten:   values.Get("flatten") == "true",
		SortBy:    TopologyQuerySortBy(values.Get("sortBy")),
		SortOrder: values.Get("sortOrder"),
		NoCache:   values.Has("noCache") || values.Has("no-cache"),
	}
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
