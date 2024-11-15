package migrate

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/flanksource/duty/functions"
	"github.com/flanksource/duty/views"
	"github.com/samber/lo"
)

func parseDependencies(script string) ([]string, error) {
	const dependencyHeader = "-- dependsOn: "

	var dependencies []string
	scanner := bufio.NewScanner(strings.NewReader(script))
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

	funcs, err := functions.GetFunctions()
	if err != nil {
		return nil, err
	}

	views, err := views.GetViews()
	if err != nil {
		return nil, err
	}

	for i, dir := range []map[string]string{funcs, views} {
		dirName := "functions"
		if i == 1 {
			dirName = "views"
		}

		for entry, content := range dir {
			path := filepath.Join(dirName, entry)
			dependents, err := parseDependencies(content)
			if err != nil {
				return nil, err
			}

			for _, dependent := range dependents {
				graph[dependent] = append(graph[dependent], strings.TrimPrefix(path, "../"))
			}
		}
	}

	return graph, nil
}
