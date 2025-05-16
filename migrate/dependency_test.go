package migrate

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestParseDependencies(t *testing.T) {
	testdata := []struct {
		script string
		want   []string
	}{
		{
			script: "-- dependsOn: a.sql,   b.sql",
			want:   []string{"a.sql", "b.sql"},
		},
		{
			script: "SELECT 1;",
			want:   nil,
		},
		{
			script: "-- dependsOn: a.sql,   b.sql,c.sql",
			want:   []string{"a.sql", "b.sql", "c.sql"},
		},
	}

	g := gomega.NewWithT(t) // use gomega with std go tests
	for _, td := range testdata {
		got, err := parseDependencies(td.script)
		if err != nil {
			t.Fatal(err.Error())
		}

		g.Expect(got).To(gomega.Equal(td.want))
	}
}

func TestDependencyMap(t *testing.T) {
	g := gomega.NewWithT(t) // use gomega with std go tests

	graph, err := getDependencyTree()
	if err != nil {
		t.Fatal(err.Error())
	}

	expected := map[string][]string{
		"functions/drop.sql":         {"views/006_config_views.sql", "views/021_notification.sql", "views/038_config_access.sql"},
		"views/006_config_views.sql": {"views/021_notification.sql"},
	}

	g.Expect(graph).To(gomega.HaveLen(len(expected)))

	for key, expectedDeps := range expected {
		g.Expect(graph).To(gomega.HaveKey(key))
		g.Expect(graph[key]).To(gomega.ConsistOf(expectedDeps))
	}
}
