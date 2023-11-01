package duty

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/testutils"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/itchyny/gojq"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDuty(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Duty Suite")
}

var postgresServer *embeddedPG.EmbeddedPostgres
var dummyData dummy.DummyData

const pgUrl = "postgres://postgres:postgres@localhost:9876/test?sslmode=disable"

func MustDB() *sql.DB {
	db, err := NewDB(pgUrl)
	if err != nil {
		panic(err)
	}
	return db
}

var _ = ginkgo.BeforeSuite(func() {
	var err error

	config, _ := testutils.GetEmbeddedPGConfig("test", 9876)
	postgresServer = embeddedPG.NewDatabase(config)
	if err = postgresServer.Start(); err != nil {
		ginkgo.Fail(err.Error())
	}
	logger.Infof("Started postgres on port 9876")

	if testutils.TestDB, testutils.TestDBPGPool, err = SetupDB(pgUrl, nil); err != nil {
		ginkgo.Fail(err.Error())
	}

	dummyData = dummy.GetStaticDummyData()
	err = dummyData.Populate(testutils.TestDB)
	Expect(err).ToNot(HaveOccurred())

	testutils.TestClient = fake.NewSimpleClientset(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"foo": []byte("secret"),
		}})
})

var _ = ginkgo.AfterSuite(func() {
	logger.Infof("Deleting dummy data")
	testDB, err := NewGorm(pgUrl, DefaultGormConfig())
	if err != nil {
		ginkgo.Fail(err.Error())
	}
	if err := dummyData.Delete(testDB); err != nil {
		ginkgo.Fail(err.Error())
	}
	logger.Infof("Stopping postgres")
	if err := postgresServer.Stop(); err != nil {
		ginkgo.Fail(err.Error())
	}
})

func readTestFile(path string) string {
	d, err := os.ReadFile(path)
	// We panic here because text fixtures should always be readable
	if err != nil {
		panic(fmt.Errorf("Unable to read file:%s due to %v", path, err))
	}
	return string(d)
}

func writeTestResult(path string, data []byte) {
	d, _ := normalizeJSON(string(data))
	_ = os.WriteFile(path+".out.json", []byte(d), 0644)
}

func match(path string, result any, jqFilter string) {
	resultJSON, err := json.Marshal(result)

	Expect(err).ToNot(HaveOccurred())

	writeTestResult(path, resultJSON)
	expected := readTestFile(path)
	matchJSON([]byte(expected), resultJSON, &jqFilter)
}

func parseJQ(v []byte, expr string) ([]byte, error) {
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

// normalizeJSON returns an indented json string.
// The keys are sorted lexicographically.
func normalizeJSON(jsonStr string) (string, error) {
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
	before, err := normalizeJSON(prevConfig)
	if err != nil {
		return "", fmt.Errorf("failed to normalize json for previous config: %w", err)
	}

	after, err := normalizeJSON(newConf)
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

func matchJSON(actual []byte, expected []byte, jqExpr *string) {
	var valueA, valueB = actual, expected
	var err error

	if jqExpr != nil {
		valueA, err = parseJQ(actual, *jqExpr)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		valueB, err = parseJQ(expected, *jqExpr)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

	}

	diff, err := generateDiff(string(valueA), string(valueB))
	Expect(err).To(BeNil())
	Expect(diff).To(BeEmpty())
}
