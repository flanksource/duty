package context

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/duration"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/models"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
)

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
		if prop.Value != nil && fmt.Sprintf("%v", prop.Default) != fmt.Sprintf("%v", prop.Value) {
			logger.Debugf("Property overridden %s=%v (default=%v)", prop.Key,
				console.Greenf("%s", nilSafe(prop.Value)),
				nilSafe(prop.Default),
			)
		}
	}
}

// Properties is a flat map of global properties (DB + CLI/env).
type Properties map[string]string

// HierarchicalProperties provides layered property lookup: CLI/env → local object annotations → parent → global DB.
type HierarchicalProperties struct {
	local  map[string]string
	parent *HierarchicalProperties
	global Properties
}

func (h HierarchicalProperties) SupportedProperties() map[string]PropertyType {
	m := make(map[string]PropertyType)
	for k, v := range supportedProperties.Items() {
		m[k] = v
	}
	return m
}

// On returns true if the property is "true", "enabled", or "on"; defaults to def if not found.
func (h HierarchicalProperties) On(def bool, keys ...string) bool {
	var v *bool
	for _, key := range keys {
		prop := PropertyType{
			Type:    "bool",
			Key:     key,
			Default: def,
		}
		if v == nil {
			k, ok := h.getProperty(key)
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

func (h HierarchicalProperties) Duration(key string, def time.Duration) time.Duration {
	if d, ok := h.getProperty(key); !ok {
		newProp(PropertyType{
			Type:    "duration",
			Key:     key,
			Default: def,
		})
		return def
	} else if dur, err := duration.ParseDuration(d); err != nil {
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
			Value:   time.Duration(dur),
		})
		return time.Duration(dur)
	}
}

func (h HierarchicalProperties) Int(key string, def int) int {
	prop := PropertyType{
		Type:    "int",
		Key:     key,
		Default: def,
	}

	if v, ok := h.getProperty(key); ok {
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

func (h HierarchicalProperties) String(key string, def string) string {
	prop := PropertyType{
		Type:    "string",
		Key:     key,
		Default: def,
	}
	if d, ok := h.getProperty(key); ok {
		prop.Value = d
		newProp(prop)
		return d
	}
	newProp(prop)
	return def
}

// Off returns true if the property is "false", "disabled", or "off".
func (h HierarchicalProperties) Off(key string, def bool) bool {
	prop := PropertyType{
		Type:    "bool",
		Key:     key,
		Default: def,
	}
	k, ok := h.getProperty(key)
	if !ok {
		newProp(prop)
		return def
	}
	v := k == "false" || k == "disabled" || k == "off"
	prop.Value = v
	newProp(prop)
	return v
}

// getProperty resolves a key with precedence: CLI/env → local → parent chain → global DB.
func (h HierarchicalProperties) getProperty(key string) (string, bool) {
	if v := properties.Get(key); v != "" {
		return v, true
	}
	if v, ok := h.local[key]; ok {
		return v, true
	}
	if h.parent != nil {
		return h.parent.getProperty(key)
	}
	v, ok := h.global[key]
	return v, ok
}

func (k Context) globalProperties() Properties {
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

	for k, v := range properties.Global.GetAll() {
		props[k] = v
	}

	propertyCache.Set("global", props, 0)
	return props
}

func extractMissionControlAnnotations(annotations map[string]string) map[string]string {
	var out map[string]string
	for key, val := range annotations {
		if after, ok := strings.CutPrefix(key, "mission-control/"); ok {
			if out == nil {
				out = make(map[string]string)
			}
			out[after] = val
		}
	}
	return out
}

// Properties returns a HierarchicalProperties that resolves keys by traversing
// the object chain (parent → child) before falling back to global DB properties.
func (k Context) Properties() HierarchicalProperties {
	global := k.globalProperties()
	root := &HierarchicalProperties{global: global}

	var cacheKeyParts []string
	current := root
	for _, o := range k.Objects() {
		meta := getObjectMeta(o)
		local := extractMissionControlAnnotations(meta.Annotations)
		if len(local) == 0 {
			continue
		}
		cacheKeyParts = append(cacheKeyParts, meta.Namespace+"/"+meta.Name)
		current = &HierarchicalProperties{local: local, parent: current}
	}

	if len(cacheKeyParts) == 0 {
		return *root
	}

	cacheKey := "properties/" + strings.Join(cacheKeyParts, ",")
	if val, ok := propertyCache.Get(cacheKey); ok {
		return val.(HierarchicalProperties)
	}
	propertyCache.Set(cacheKey, *current, 0)
	return *current
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
