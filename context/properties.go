package context

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/models"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/patrickmn/go-cache"
)

var Local map[string]string
var supportedProperties = cmap.New[string]()

var propertyCache = cache.New(time.Minute*15, time.Minute*15)

func (k Context) ClearCache() {
	propertyCache = cache.New(time.Minute*15, time.Minute*15)
}

func nilSafe(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
func newProp(key, def string, val interface{}) {
	if loaded := supportedProperties.SetIfAbsent(key, fmt.Sprintf("%s", val)); loaded {
		if val == nil {
			logger.Tracef("property: %s=%v", key, console.Grayf(nilSafe(def)))
		} else {
			logger.Debugf("property: %s=%v (default %v)", key, console.Greenf("%s", val), nilSafe(def))
		}
	}
}

func (p Properties) SupportedProperties() map[string]string {
	m := make(map[string]string)
	for t := range supportedProperties.IterBuffered() {
		m[t.Key] = nilSafe(t.Val)
	}
	return m
}

type Properties map[string]string

// Returns true if the property is true|enabled|on, if there is no property it defaults to true
func (p Properties) On(def bool, keys ...string) bool {
	for _, key := range keys {
		k, ok := p[key]
		if ok {
			v := k == "true" || k == "enabled" || k == "on"
			newProp(key, fmt.Sprintf("%v", def), v)
			return v
		}
		newProp(key, fmt.Sprintf("%v", def), nil)
	}
	return def
}

func (p Properties) Duration(key string, def time.Duration) time.Duration {
	if d, ok := p[key]; !ok {
		newProp(key, fmt.Sprintf("%v", def), nil)
		return def
	} else if dur, err := time.ParseDuration(d); err != nil {
		logger.Warnf("property[%s] invalid duration %s", key, d)
		return def
	} else {
		newProp(key, fmt.Sprintf("%v", def), dur)
		return dur
	}
}

func (p Properties) Int(key string, def int) int {
	if d, ok := p[key]; !ok {
		newProp(key, fmt.Sprintf("%v", def), nil)
		return def
	} else if i, err := strconv.Atoi(d); err != nil {
		logger.Warnf("property[%s] invalid int %s", key, d)
		return def
	} else {
		newProp(key, fmt.Sprintf("%v", def), i)
		return i
	}
}

func (p Properties) String(key string, def string) string {
	if d, ok := p[key]; ok {
		newProp(key, fmt.Sprintf("%v", def), d)
		return d
	}
	newProp(key, fmt.Sprintf("%v", def), nil)
	return def

}

// Returns true if the property is false|disabled|off, if there is no property it defaults to true
func (p Properties) Off(key string, def bool) bool {
	k, ok := p[key]
	if !ok {
		newProp(key, fmt.Sprintf("%v", def), nil)
		return def
	}
	v := k == "false" || k == "disabled" || k == "off"
	newProp(key, fmt.Sprintf("%v", def), v)
	return v
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
			logger.Infof("%s(local)=%s", prop.Name, prop.Value)
			props[prop.Name] = prop.Value
		}
	}

	for k, v := range Local {
		props[k] = v
	}

	propertyCache.Set("global", props, 0)
	return props
}

func SetLocalProperty(property, value string) {
	if Local == nil {
		Local = make(map[string]string)
	}

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
