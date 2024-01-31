package types

import (
	"encoding/json"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// asMap marshals the given struct into a map.
func asMap(t any, removeFields ...string) map[string]any {
	m := make(map[string]any)
	b, _ := json.Marshal(&t)
	if err := json.Unmarshal(b, &m); err != nil {
		return m
	}

	for _, field := range removeFields {
		delete(m, field)
	}

	return m
}

type Items []string

func (items Items) String() string {
	return strings.Join(items, ",")
}

// contains returns true if any of the items in the list match the item
// negative matches are supported by prefixing the item with a !
// * matches everything
func (items Items) Contains(item string) bool {
	if len(items) == 0 {
		return true
	}

	negations := 0
	for _, i := range items {
		if strings.HasPrefix(i, "!") {
			negations++
			if item == strings.TrimPrefix(i, "!") {
				return false
			}
		}
	}

	if negations == len(items) {
		// none of the negations matched
		return true
	}

	for _, i := range items {
		if strings.HasPrefix(i, "!") {
			continue
		}
		if i == "*" || item == i {
			return true
		}
	}
	return false
}

func (items Items) WithNegation() []string {
	var result []string
	for _, item := range items {
		if strings.HasPrefix(item, "!") {
			result = append(result, item[1:])
		}
	}
	return result
}

// Sort returns a sorted copy
func (items Items) Sort() Items {
	copy := items
	sort.Slice(copy, func(i, j int) bool { return items[i] < items[j] })
	return copy
}

func (items Items) WithoutNegation() []string {
	var result []string
	for _, item := range items {
		if !strings.HasPrefix(item, "!") {
			result = append(result, item)
		}
	}
	return result
}

func (items Items) Where(query *gorm.DB, col string) *gorm.DB {
	if items == nil {
		return query
	}

	negated := items.WithNegation()
	if len(negated) > 0 {
		query = query.Where("NOT "+col+" IN ?", negated)
	}

	positive := items.WithoutNegation()
	if len(positive) > 0 {
		query = query.Where(col+" IN ?", positive)
	}

	return query
}
