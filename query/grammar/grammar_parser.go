//go:generate go run github.com/mna/pigeon@v1.3.0 -o grammar.go grammar.peg
package grammar

import (
	"fmt"
	"strings"

	"github.com/flanksource/commons/logger"
)

type Source struct {
	Name string   `json:"name,omitempty"`
	Path []string `json:"path,omitempty"`
}

type NumberUnit struct {
	Number interface{} `json:"number,omitempty"` // int64/float64
	Units  string      `json:"units,omitempty"`
}

func makeSource(name interface{}, path interface{}) (string, error) {
	ps := path.([]interface{})

	paths := make([]string, 0)
	for _, p := range ps {
		pa := p.([]interface{})
		px := pa[1:]
		for _, pi := range px {
			paths = append(paths, pi.(string))
		}
	}
	return strings.Join(append([]string{name.(string)}, paths...), "."), nil
}

func makeFQFromQuery(a interface{}) (interface{}, error) {
	return a.(*QueryField), nil
}

//nolint:unused
func makeCatchAll(f interface{}) (*QueryField, error) {
	logger.Warnf("ctach all %v (%T)", f, f)

	switch v := f.(type) {
	case string:
		return &QueryField{Op: "rest", Value: v}, nil
	case []byte:
		return &QueryField{Op: "rest", Value: string(v)}, nil
	case []interface{}:

		rest := ""
		for _, i := range v {
			rest += fmt.Sprintf("%s", i)
		}
		return &QueryField{Op: "rest", Value: rest}, nil
	}
	return &QueryField{Op: "rest", Value: f}, nil
}

func makeFQFromField(f interface{}) (*QueryField, error) {
	return f.(*QueryField), nil
}

//nolint:unused
func makeQuery(a, b interface{}) (*QueryField, error) {
	q := &QueryField{
		Op: "or",
	}

	switch v := a.(type) {
	case *QueryField:
		q.Fields = append(q.Fields, v)
	default:
		logger.Warnf("Unknown type for query.a: %v = %T", a, a)
	}

	switch v := b.(type) {
	case *QueryField:
		q.Fields = append(q.Fields, v)
	case []interface{}:
		for _, i := range v {
			switch v2 := i.(type) {
			case *QueryField:
				q.Fields = append(q.Fields, v2)

			default:
				logger.Warnf("Unknown array item: %v (%T)", i, i)
			}
		}
	default:
		logger.Warnf("Unknown type for query.b: %v = %T", b, b)
	}

	return q, nil
}

func makeAndQuery(a any, b any) (*QueryField, error) {
	q := &QueryField{Op: "and"}

	switch v := a.(type) {
	case *QueryField:
		q.Fields = append(q.Fields, v)

	default:
		logger.Warnf("Unknown type for a: %v = %T", a, a)
	}

	switch v := b.(type) {
	case *QueryField:
		q.Fields = append(q.Fields, v)
	case []interface{}:
		for _, i := range v {
			switch v2 := i.(type) {
			case *QueryField:
				q.Fields = append(q.Fields, v2)
			default:
				logger.Warnf("Unknown array item: %v (%T)", i, i)
			}
		}
	default:
		logger.Warnf("Unknown type for b: %v = %T", b, b)
	}

	return q, nil
}

func makeValue(val interface{}) (interface{}, error) {
	return val, nil
}

func makeMeasure(num interface{}, units interface{}) (*NumberUnit, error) {
	retVal := &NumberUnit{Number: num, Units: units.(string)}

	return retVal, nil
}

func stringFromChars(chars interface{}) string {
	str := ""
	r := chars.([]interface{})
	for _, i := range r {
		j := i.([]uint8)
		str += string(j[0])
	}
	return str
}

func FlatFields(qf *QueryField) []string {
	var fields []string
	if qf.Field != "" {
		fields = append(fields, qf.Field)
	}
	for _, f := range qf.Fields {
		fields = append(fields, FlatFields(f)...)
	}
	return fields
}

func ParsePEG(peg string) (*QueryField, error) {
	stats := Stats{}

	v, err := Parse("", []byte(peg), Statistics(&stats, "no match"))
	if err != nil {
		return nil, fmt.Errorf("error parsing peg: %w", err)
	}

	rv, ok := v.(*QueryField)
	if !ok {
		return nil, fmt.Errorf("return type not QueryField")
	}

	return rv, nil
}
