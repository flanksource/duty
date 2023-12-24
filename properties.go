package duty

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/flanksource/duty/context"
)

func UpdateProperty(ctx context.Context, key, value string) error {
	query := "INSERT INTO properties (name, value) VALUES (?,?) ON CONFLICT (name) DO UPDATE SET value = excluded.value"
	defer ctx.ClearCache()
	return ctx.DB().Exec(query, key, value).Error
}

func UpdateProperties(ctx context.Context, props map[string]string) error {
	var values []string
	var args []interface{}
	for key, value := range props {
		values = append(values, "(?, ?)")
		args = append(args, key, value)
	}

	if len(values) == 0 {
		return nil
	}
	query := fmt.Sprintf("INSERT INTO properties (name, value) VALUES %s ON CONFLICT (name) DO UPDATE SET value = excluded.value", strings.Join(values, ","))
	defer ctx.ClearCache()
	return ctx.DB().Exec(query, args...).Error
}

func UpdatePropertiesFromFile(ctx context.Context, filename string) error {
	props, err := ParsePropertiesFile(filename)
	if err != nil {
		return err
	}
	return UpdateProperties(ctx, props)
}

func ParsePropertiesFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
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
