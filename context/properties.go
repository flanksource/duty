package context

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/duration"
	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty/models"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/patrickmn/go-cache"
)

var supportedProperties = cmap.New[PropertyType]()

var propertyCache = cache.New(time.Minute*15, time.Minute*15)

type PropertyType struct {
	Key     string `json:"-"`
	Value   any    `json:"value,omitempty"`
	Default any    `json:"default,omitempty"`
	Type    string `json:"type,omitempty"`
}

func (k Context) ClearCache() {
	propertyCache = cache.New(time.Minute*15, time.Minute*15)
}

func nilSafe(values ...any) string {
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
	maps.Copy(m, supportedProperties.Items())
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
				v = new(k == "true" || k == "enabled" || k == "on")
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

// WithPrefix returns all properties whose key begins with prefix, resolved
// with the normal precedence (CLI/env → local → parent chain → global DB).
// The returned keys have the prefix STRIPPED. Later sources in the chain
// override earlier ones, matching getProperty's semantics.
func (h HierarchicalProperties) WithPrefix(prefix string) map[string]string {
	out := make(map[string]string)

	// Walk global first (lowest precedence).
	for k, v := range h.global {
		if after, ok := strings.CutPrefix(k, prefix); ok {
			out[after] = v
		}
	}

	// Then walk the parent chain from root to this node (so this node
	// overrides its ancestors).
	var chain []HierarchicalProperties
	cur := &h
	for cur != nil {
		chain = append(chain, *cur)
		cur = cur.parent
	}
	for i := len(chain) - 1; i >= 0; i-- {
		for k, v := range chain[i].local {
			if after, ok := strings.CutPrefix(k, prefix); ok {
				out[after] = v
			}
		}
	}

	// Finally, CLI/env takes highest precedence.
	for k, v := range properties.Global.GetAll() {
		if after, ok := strings.CutPrefix(k, prefix); ok {
			out[after] = v
		}
	}

	return out
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

	maps.Copy(props, properties.Global.GetAll())

	propertyCache.Set("global", props, 0)
	return props
}

func extractAnnotations(annotations map[string]string) map[string]string {
	var out map[string]string
	for key, val := range annotations {
		for _, prefix := range annotationPrefixes {
			if prefix == "" {
				continue
			}
			if after, ok := strings.CutPrefix(key, prefix); ok {
				if out == nil {
					out = make(map[string]string)
				}
				out[after] = val
			}
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
		local := extractAnnotations(meta.Annotations)
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
	var args []any
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

const HARMaxBodySizeDefault = 64 * 1024

func (k Context) EffectiveLogLevel(feature string) (logger.LogLevel, string) {
	return k.effectiveObservabilityLevel(feature, false)
}

func (k Context) EffectiveHARLevel(feature string) (logger.LogLevel, string) {
	return k.effectiveObservabilityLevel(feature, true)
}

func (k Context) IsHARCaptureEnabled(feature string) bool {
	level, _ := k.EffectiveHARLevel(feature)
	return level >= logger.Debug
}

func (k Context) IsHTTPLoggingEnabled(feature string) bool {
	level, _ := k.EffectiveLogLevel(feature)
	return level >= logger.Debug
}

func (k Context) HTTPLoggingContent(feature string) (headers bool, bodies bool) {
	level, _ := k.EffectiveLogLevel(feature)
	return level >= logger.Debug, level >= logger.Trace
}

func (k Context) HARConfig(feature string) har.HARConfig {
	cfg := har.DefaultConfig()
	cfg.MaxBodySize = int64(k.Properties().Int("har.maxBodySize", HARMaxBodySizeDefault))
	if v := k.Properties().String("har.captureContentTypes", ""); v != "" {
		cfg.CaptureContentTypes = splitCSV(v)
	}
	return cfg
}

func (k Context) EffectiveHARCollector(feature string, explicit *har.Collector) *har.Collector {
	if explicit != nil {
		explicit.Config = k.HARConfig(feature)
		return explicit
	}
	level, _ := k.EffectiveHARLevel(feature)
	if level < logger.Debug {
		return nil
	}
	collector := k.HARCollector()
	if collector != nil {
		collector.Config = k.HARConfig(feature)
	}
	return collector
}

func (k Context) effectiveObservabilityLevel(feature string, harCapture bool) (logger.LogLevel, string) {
	feature = strings.TrimSpace(strings.ToLower(feature))
	if feature == "" {
		feature = "http"
	}

	level := normalizeFeatureLevel(k.Logger.GetLevel())
	if std := logger.StandardLogger(); std != nil {
		level = maxLevel(level, std.GetLevel())
	}
	source := "logger"
	props := k.Properties()

	add := func(candidate logger.LogLevel, candidateSource string) {
		candidate = normalizeFeatureLevel(candidate)
		if candidate > level {
			level = candidate
			source = candidateSource
		}
	}
	addProperty := func(key string) {
		if v := props.String(key, ""); v != "" {
			add(logger.ParseLevel(k.Logger, v), key)
		}
	}
	addAnnotation := func(key string) {
		for _, o := range k.Objects() {
			annotations := getObjectMeta(o).Annotations
			if len(annotations) == 0 {
				continue
			}
			if v := annotationValue(annotations, key); v != "" {
				add(logger.ParseLevel(k.Logger, v), "annotation:"+key)
			}
		}
	}

	addProperty("log.level")
	addAnnotation("log.level")
	for _, o := range k.Objects() {
		annotations := getObjectMeta(o).Annotations
		if len(annotations) == 0 {
			continue
		}
		if annotationValue(annotations, "trace") == "true" {
			add(logger.Trace, "annotation:trace")
		} else if annotationValue(annotations, "debug") == "true" {
			add(logger.Debug, "annotation:debug")
		}
	}

	if harCapture {
		addProperty("log.level.http.har")
		for _, f := range featureLevelKeys(feature) {
			addProperty("log.level." + f + ".har")
		}
		addAnnotation("log.level.http.har")
		for _, f := range featureLevelKeys(feature) {
			addAnnotation("log.level." + f + ".har")
		}
	} else {
		addProperty("log.level.http")
		for _, f := range featureLevelKeys(feature) {
			addProperty("log.level." + f)
		}
		addAnnotation("log.level.http")
		for _, f := range featureLevelKeys(feature) {
			addAnnotation("log.level." + f)
		}
	}

	return level, source
}

func normalizeFeatureLevel(level logger.LogLevel) logger.LogLevel {
	if level == logger.Silent || level < logger.Info {
		return logger.Info
	}
	return level
}

func maxLevel(a, b logger.LogLevel) logger.LogLevel {
	a = normalizeFeatureLevel(a)
	b = normalizeFeatureLevel(b)
	if a > b {
		return a
	}
	return b
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func featureLevelKeys(feature string) []string {
	if feature == "http" {
		return nil
	}
	switch feature {
	case "kubernetes":
		return []string{"kubernetes", "kubectl", "k8s"}
	default:
		return []string{feature}
	}
}
