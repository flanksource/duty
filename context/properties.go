package context

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/models"
	"github.com/patrickmn/go-cache"
)

var Local map[string]string

var propertyCache = cache.New(time.Minute*15, time.Minute*15)

func (k Context) ClearCache() {
	propertyCache = cache.New(time.Minute*15, time.Minute*15)
}

type Properties map[string]string

func (p Properties) On(key string) bool {
	return p[key] == "true" || p[key] == "off"
}

func (p Properties) Off(key string) bool {
	return p[key] == "false" || p[key] == "disabled"
}

// Properties returns a cached map of properties
func (k Context) Properties() Properties {
	// properties are currently global, but in future we might have context specific properties as well
	if val, ok := propertyCache.Get("global"); ok {
		return val.(map[string]string)
	}

	var props = make(map[string]string)
	if k.DB() != nil {
		var rows []models.AppProperty
		if err := k.DB().Find(&rows).Error; err != nil {
			return props
		}

		for _, prop := range rows {
			props[prop.Name] = prop.Value
		}
	}

	for k, v := range Local {
		props[k] = v
	}

	propertyCache.Set("global", props, 0)
	return props
}

func SetLocalProperty(ctx Context, property, value string) {
	Local[property] = value
}

func UpdateProperty(ctx Context, key, value string) error {
	query := "INSERT INTO properties (name, value) VALUES (?,?) ON CONFLICT (name) DO UPDATE SET value = excluded.value"
	logger.Debugf("Updated property %s = %s", key, value)
	defer ctx.ClearCache()
	return ctx.DB().Exec(query, key, value).Error
}

func UpdateProperties(ctx Context, props map[string]string) error {
	var values []string
	var args []interface{}
	for key, value := range props {
		values = append(values, "(?, ?)")
		args = append(args, key, value)
		logger.Debugf("Updated property %s = %s", key, value)
	}

	if len(values) == 0 {
		return nil
	}
	query := fmt.Sprintf("INSERT INTO properties (name, value) VALUES %s ON CONFLICT (name) DO UPDATE SET value = excluded.value", strings.Join(values, ","))
	defer ctx.ClearCache()
	return ctx.DB().Exec(query, args...).Error
}

func LoadPropertiesFromFile(ctx Context, filename string) error {
	logger.Infof("Loading properties from %s", filename)
	props, err := ParsePropertiesFile(filename)
	if err != nil {
		return err
	}
	Local = props
	for k, v := range Local {
		logger.Infof("%s(local)=%s", k, v)
	}
	defer ctx.ClearCache()
	return nil
}

func ParsePropertiesFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	var props = make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}

		key := strings.TrimSpace(tokens[0])
		value := strings.TrimSpace(tokens[1])
		props[key] = value
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return props, nil
}
