package types

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/flanksource/commons/logger"
	"gorm.io/gorm"
)

// marshalIgnoringOmitempty marshals a struct to JSON without considering omitempty tags.
func marshalIgnoringOmitempty(v any) ([]byte, error) {
	// Get the reflect type of the value
	t := reflect.TypeOf(v)

	// Create a new value of the same type
	newValue := reflect.New(t).Elem()

	// Copy the original value to the new value
	newValue.Set(reflect.ValueOf(v))
	structType := reflect.TypeOf(v)
	newFields := make([]reflect.StructField, structType.NumField())
	// Iterate over the fields of the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Remove the omitempty tag from the JSON tag
		field.Tag = reflect.StructTag(strings.Replace(string(field.Tag), ",omitempty", "", -1))

		// Set the modified tag back to the field
		newFields[i] = field
	}
	newStructType := reflect.StructOf(newFields)

	// Create an instance of the new struct type
	newStruct := reflect.New(newStructType).Elem()

	// Copy values from the original struct to the new one
	originalStruct := reflect.ValueOf(v)
	for i := 0; i < structType.NumField(); i++ {
		newStruct.Field(i).Set(originalStruct.Field(i))
	}

	// Marshal the modified value to JSON
	return json.Marshal(newStruct.Interface())
}

// AsMap marshals the given struct into a map.
func AsMap(t any, removeFields ...string) map[string]any {
	m := make(map[string]any)
	b, err := marshalIgnoringOmitempty(t)
	if err != nil {
		logger.Infof("ERROR %v", err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m
	}

	logger.Infof("m is %v", m)
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
