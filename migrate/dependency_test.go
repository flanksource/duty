package migrate

import (
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseDependencies(t *testing.T) {
	testdata := []struct {
		data io.ReadCloser
		want []string
	}{
		{
			data: io.NopCloser(strings.NewReader("-- dependsOn: a.sql,   b.sql")),
			want: []string{"a.sql", "b.sql"},
		},
		{
			data: io.NopCloser(strings.NewReader("SELECT 1;")),
			want: nil,
		},
		{
			data: io.NopCloser(strings.NewReader("-- dependsOn: a.sql,   b.sql,c.sql")),
			want: []string{"a.sql", "b.sql", "c.sql"},
		},
	}

	for _, td := range testdata {
		got, err := parseDependencies(td.data)
		if err != nil {
			t.Fatal(err.Error())
		}

		if diff := cmp.Diff(got, td.want); diff != "" {
			t.Fatalf("%s", diff)
		}
	}
}

func TestDependencyMap(t *testing.T) {
	graph, err := getDependencyTree()
	if err != nil {
		t.Fatal(err.Error())
	}

	if diff := cmp.Diff(graph["functions/drop.sql"], []string{"views/021_notification.sql"}); diff != "" {
		t.Fatalf("%v", diff)
	}
}
