package views

import (
	"embed"

	"github.com/flanksource/commons/properties"
)

//go:embed *.sql
var views embed.FS

func GetViews() (map[string]string, error) {
	scripts, err := views.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var funcs = make(map[string]string)
	for _, file := range scripts {
		script, err := views.ReadFile(file.Name())
		if err != nil {
			return nil, err
		}
		funcs[file.Name()] = string(script)
	}

	usePrecomputed := properties.On(false, "rls.precomputed_scope")
	if precomputed, ok := funcs["9998_rls_enable_precomputed.sql"]; ok {
		if usePrecomputed {
			funcs["9998_rls_enable.sql"] = precomputed
		}
		delete(funcs, "9998_rls_enable_precomputed.sql")
	}
	return funcs, nil
}
