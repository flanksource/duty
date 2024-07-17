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
	"github.com/samber/lo"
)

var Local map[string]string
var supportedProperties = cmap.New[PropertyType]()

var propertyCache = cache.New(time.Minute*15, time.Minute*15)

type PropertyType struct {
	Key     string      `json:"-"`
	Value   interface{} `json:"value,omitempty"`
	Default interface{} `json:"default,omitempty"`
	Type    string      `json:"type,omitempty"`
}

func (k Context) ClearCache() {
	propertyCache = cache.New(time.Minute*15, time.Minute*15)
}

func nilSafe(values ...interface{}) string {
	for _, v := range values {
		if v != nil && v != "" {
			switch t := v.(type) {
			case *bool:
				return fmt.Sprintf("%v", *t)
			default:
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ""
}

func newProp(prop PropertyType) {
	if loaded := supportedProperties.SetIfAbsent(prop.Key, prop); loaded {
		if prop.Value != nil && prop.Default != prop.Value {
			logger.Debugf("Property overridden %s=%v (default=%v)", prop.Key, console.Greenf(nilSafe(prop.Value)), nilSafe(prop.Default))
		}
	}
}
func (p Properties) SupportedProperties() map[string]PropertyType {
	m := make(map[string]PropertyType)
	for t := range supportedProperties.IterBuffered() {
		m[t.Key] = t.Val
	}
	return m
}

type Properties map[string]string

// Returns true if the property is true|enabled|on, if there is no property it defaults to true
func (p Properties) On(def bool, keys ...string) bool {
	var v *bool
	for _, key := range keys {
		prop := PropertyType{
			Type:    "bool",
			Key:     key,
			Default: def,
		}
		if v == nil {
			k, ok := p[key]
			if ok {
				v = lo.ToPtr(k == "true" || k == "enabled" || k == "on")
				prop.Value = v
			}
		}
		newProp(prop)
	}
	if v != nil {
		return *v
	}
	return def
}

func (p Properties) Duration(key string, def time.Duration) time.Duration {
	if d, ok := p[key]; !ok {
		newProp(PropertyType{
			Type:    "duration",
			Key:     key,
			Default: def,
		})
		return def
	} else if dur, err := time.ParseDuration(d); err != nil {
		newProp(PropertyType{
			Type:    "duration",
			Key:     key,
			Default: def,
			Value:   d,
		})
		logger.Warnf("property[%s] invalid duration %s", key, d)
		return def
	} else {
		newProp(PropertyType{
			Type:    "duration",
			Key:     key,
			Default: def,
			Value:   dur,
		})
		return dur
	}
}

func (p Properties) Int(key string, def int) int {
	prop := PropertyType{
		Type:    "int",
		Key:     key,
		Default: def,
	}

	if v, ok := p[key]; ok {
		prop.Value = v
		if i, err := strconv.Atoi(v); err != nil {
			logger.Warnf("property[%s] invalid int %s", key, v)
		} else {
			prop.Value = i
			newProp(prop)
			return i
		}
	}
	newProp(prop)
	return def

}

func (p Properties) String(key string, def string) string {
	prop := PropertyType{
		Type:    "string",
		Key:     key,
		Default: def,
	}
	if d, ok := p[key]; ok {
		prop.Value = d
		newProp(prop)
		return d
	}
	newProp(prop)
	return def

}

// Returns true if the property is false|disabled|off, if there is no property it defaults to true
func (p Properties) Off(key string, def bool) bool {

	prop := PropertyType{
		Type:    "bool",
		Key:     key,
		Default: def,
	}
	k, ok := p[key]
	if !ok {
		newProp(prop)
		return def
	}
	v := k == "false" || k == "disabled" || k == "off"
	prop.Value = v
	newProp(prop)
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
