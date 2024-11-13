package migrate

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
)

func parseDependencies(f io.ReadCloser) ([]string, error) {
	defer f.Close()

	const dependencyHeader = "-- dependsOn: "
	var dependencies []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, dependencyHeader) {
			break
		}

		line = strings.TrimPrefix(line, dependencyHeader)
		deps := strings.Split(line, ",")
		dependencies = append(dependencies, lo.Map(deps, func(x string, _ int) string {
			return strings.TrimSpace(x)
		})...)
	}

	return dependencies, nil
}

// DependencyMap map holds path -> dependents
type DependencyMap map[string][]string

// getDependencyTree returns a list of scripts and its dependents
//
// example: if a.sql dependsOn b.sql, c.sql
// it returns
//
//	{
//		b.sql: []string{a.sql},
//		c.sql: []string{a.sql},
//	}
func getDependencyTree() (DependencyMap, error) {
	graph := make(DependencyMap)

	dirs := []string{"../functions", "../views"}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %w", err)
			}

			dependents, err := parseDependencies(f)
			if err != nil {
				return nil, err
			}

			for _, depedent := range dependents {
				graph[depedent] = append(graph[depedent], strings.TrimPrefix(path, "../"))
			}
		}
	}

	return graph, nil
}
