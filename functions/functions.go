package functions

import "embed"

//go:embed *.sql
var functions embed.FS

var Functions map[string]string

func GetFunctions() (map[string]string, error) {
	scripts, err := functions.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var funcs = make(map[string]string)
	for _, file := range scripts {
		script, err := functions.ReadFile(file.Name())
		if err != nil {
			return nil, err
		}
		funcs[file.Name()] = string(script)
	}
	return funcs, nil
}
