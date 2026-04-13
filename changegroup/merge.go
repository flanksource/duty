package changegroup

import (
	"fmt"
	"reflect"
	"time"

	"github.com/flanksource/duty/types"
)

// Merge combines new typed group details into the existing stored value,
// using `merge:"..."` struct tags on the concrete GroupType to decide per-field
// strategy. Both arguments must be non-nil and of the same concrete type.
//
// Strategies:
//   - append:    dedupe-append slices.
//   - firstSet:  keep the existing value if it was already set; otherwise take new.
//   - min / max: reduce on comparable scalars (time.Time, numbers, strings).
//   - (default): last-write-wins for scalars; new slice/map replaces existing
//     iff the new value is non-empty, otherwise the old value is preserved.
//
// Nil pointers / zero slices on the "new" side never clobber a populated
// existing value — this lets CEL omit fields a given member doesn't know about.
func Merge(existing, incoming types.GroupType) (types.GroupType, error) {
	if existing == nil {
		return incoming, nil
	}
	if incoming == nil {
		return existing, nil
	}
	if existing.Kind() != incoming.Kind() {
		return nil, fmt.Errorf("changegroup: cannot merge %q with %q", existing.Kind(), incoming.Kind())
	}

	// Work on addressable copies so reflect.Set works even when callers pass
	// value types (e.g. types.DeploymentGroup{}).
	ex := reflect.New(reflect.TypeOf(existing)).Elem()
	ex.Set(reflect.ValueOf(existing))
	in := reflect.ValueOf(incoming)

	if ex.Kind() != reflect.Struct {
		return nil, fmt.Errorf("changegroup: cannot merge non-struct GroupType %q", existing.Kind())
	}

	for i := 0; i < ex.NumField(); i++ {
		field := ex.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		strategy := field.Tag.Get("merge")
		exField := ex.Field(i)
		inField := in.Field(i)
		mergeField(exField, inField, strategy)
	}

	return ex.Interface().(types.GroupType), nil
}

func mergeField(dst, src reflect.Value, strategy string) {
	// Never clobber with an explicit zero value on the incoming side — this
	// lets rules omit fields they don't care about.
	if isZero(src) {
		return
	}

	switch strategy {
	case "append":
		mergeAppend(dst, src)
	case "firstSet":
		if isZero(dst) {
			dst.Set(src)
		}
	case "min":
		if isZero(dst) || lessThan(src, dst) {
			dst.Set(src)
		}
	case "max":
		if isZero(dst) || lessThan(dst, src) {
			dst.Set(src)
		}
	case "mapMerge":
		mergeMap(dst, src)
	default:
		// last-write-wins for scalars; replace for slices/maps when src is non-empty.
		dst.Set(src)
	}
}

// mergeMap performs a per-key merge of two maps of the same type. Keys only
// in dst are preserved; keys in both take the src value (last-write-wins).
func mergeMap(dst, src reflect.Value) {
	if dst.Kind() != reflect.Map || src.Kind() != reflect.Map {
		dst.Set(src)
		return
	}
	if dst.IsNil() {
		dst.Set(reflect.MakeMapWithSize(dst.Type(), src.Len()))
	}
	iter := src.MapRange()
	for iter.Next() {
		dst.SetMapIndex(iter.Key(), iter.Value())
	}
}

// mergeAppend performs a dedupe-append of src slice elements into dst.
// Both must be slices of the same element kind.
func mergeAppend(dst, src reflect.Value) {
	if dst.Kind() != reflect.Slice || src.Kind() != reflect.Slice {
		// Mis-tagged field; fall back to replace.
		dst.Set(src)
		return
	}
	seen := make(map[any]struct{}, dst.Len()+src.Len())
	result := reflect.MakeSlice(dst.Type(), 0, dst.Len()+src.Len())
	for i := 0; i < dst.Len(); i++ {
		v := dst.Index(i).Interface()
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = reflect.Append(result, dst.Index(i))
	}
	for i := 0; i < src.Len(); i++ {
		v := src.Index(i).Interface()
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = reflect.Append(result, src.Index(i))
	}
	dst.Set(result)
}

// isZero reports whether v holds its type's zero value. For pointers, a nil
// pointer is zero; for slices/maps, len == 0 is zero.
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

// lessThan compares two values of the same comparable type. It supports
// time.Time, signed/unsigned integers, floats and strings. For any other
// type it returns false (conservative: don't overwrite).
func lessThan(a, b reflect.Value) bool {
	// Dereference pointers.
	for a.Kind() == reflect.Ptr {
		if a.IsNil() {
			return true
		}
		a = a.Elem()
	}
	for b.Kind() == reflect.Ptr {
		if b.IsNil() {
			return false
		}
		b = b.Elem()
	}

	if t, ok := a.Interface().(time.Time); ok {
		if u, ok := b.Interface().(time.Time); ok {
			return t.Before(u)
		}
	}

	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return a.Uint() < b.Uint()
	case reflect.Float32, reflect.Float64:
		return a.Float() < b.Float()
	case reflect.String:
		return a.String() < b.String()
	}
	return false
}

