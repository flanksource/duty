package views

import "embed"

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
	return funcs, nil
}
