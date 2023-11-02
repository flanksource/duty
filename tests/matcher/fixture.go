package matcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/itchyny/gojq"
	. "github.com/onsi/gomega"
)

func MatchFixture(path string, result any, jqFilter string) {
	resultJSON, err := json.Marshal(result)

	Expect(err).ToNot(HaveOccurred())

	writeTestResult(path, resultJSON)
	expected := readTestFile(path)
	CompareJSON([]byte(expected), resultJSON, &jqFilter)
}

func readTestFile(p string) string {
	dir, _ := os.Getwd()
	p = path.Join(dir, p)
	d, err := os.ReadFile(p)
	// We panic here because text fixtures should always be readable
	if err != nil {
		return "{}"
	}
	return string(d)
}

func writeTestResult(path string, data []byte) {
	d, _ := NormalizeJSON(string(data))
	if err := os.WriteFile(path+".out.json", []byte(d), 0644); err != nil {
		panic(err)
	}

}

func ParseJQ(v []byte, expr string) ([]byte, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, err
	}
	var input any
	err = json.Unmarshal(v, &input)
	if err != nil {
		return nil, err
	}
	iter := query.Run(input)
	var jsonVal []byte
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := val.(error); ok {
			return nil, fmt.Errorf("Error parsing jq: %v", err)
		}

		jsonVal, err = json.Marshal(val)
		if err != nil {
			return nil, err
		}
	}
	return jsonVal, nil
}

// NormalizeJSON returns an indented json string.
// The keys are sorted lexicographically.
func NormalizeJSON(jsonStr string) (string, error) {
	var jsonStrMap interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonStrMap); err != nil {
		return "", err
	}

	jsonStrIndented, err := json.MarshalIndent(jsonStrMap, "", "\t")
	if err != nil {
		return "", err
	}

	return string(jsonStrIndented), nil
}

// generateDiff calculates the diff (git style) between the given 2 configs.
func generateDiff(newConf, prevConfig string) (string, error) {
	// We want a nicely indented json config with each key-vals in new line
	// because that gives us a better diff. A one-line json string config produces diff
	// that's not very helpful.
	before, err := NormalizeJSON(prevConfig)
	if err != nil {
		return "", fmt.Errorf("failed to normalize json for previous config: %w", err)
	}

	after, err := NormalizeJSON(newConf)
	if err != nil {
		return "", fmt.Errorf("failed to normalize json for new config: %w", err)
	}

	edits := myers.ComputeEdits("", before, after)
	if len(edits) == 0 {
		return "", nil
	}

	diff := fmt.Sprint(gotextdiff.ToUnified("before", "after", before, edits))
	return diff, nil
}

func CompareJSON(actual []byte, expected []byte, jqExpr *string) {
	var valueA, valueB = actual, expected
	var err error

	if jqExpr != nil {
		valueA, err = ParseJQ(actual, *jqExpr)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		valueB, err = ParseJQ(expected, *jqExpr)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

	}

	diff, err := generateDiff(string(valueA), string(valueB))
	Expect(err).To(BeNil())
	Expect(diff).To(BeEmpty())
}
