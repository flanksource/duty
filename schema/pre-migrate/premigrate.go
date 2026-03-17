package premigrate

import "embed"

//go:embed *.sql
var Premigrations embed.FS

func GetPremigrations() (map[string]string, error) {
	scripts, err := Premigrations.ReadDir(".")
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, file := range scripts {
		script, err := Premigrations.ReadFile(file.Name())
		if err != nil {
			return nil, err
		}
		result[file.Name()] = string(script)
	}
	return result, nil
}
