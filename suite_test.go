package duty

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/testutils"
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

const pgUrl = "postgres://postgres:postgres@localhost:9876/test?sslmode=disable"

func MustDB() *sql.DB {
	db, err := NewDB(pgUrl)
	if err != nil {
		panic(err)
	}
	return db
}

var _ = ginkgo.BeforeSuite(func() {
	postgresServer = embeddedPG.NewDatabase(embeddedPG.DefaultConfig().
		Database("test").
		Port(9876).
		Logger(io.Discard))
	if err := postgresServer.Start(); err != nil {
		ginkgo.Fail(err.Error())
	}
	logger.Infof("Started postgres on port 9876")
	var err error
	if testutils.TestDBPGPool, err = NewPgxPool(pgUrl); err != nil {
		ginkgo.Fail(err.Error())
	}
	if _, err := NewDB(pgUrl); err != nil {
		ginkgo.Fail(err.Error())
	}
	err = Migrate(pgUrl)
	Expect(err).ToNot(HaveOccurred())

	testutils.TestDB, err = NewGorm(pgUrl, DefaultGormConfig())
	Expect(err).ToNot(HaveOccurred())
	Expect(testutils.TestDB).ToNot(BeNil())

	err = dummy.PopulateDBWithDummyModels(testutils.TestDB)
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
	if err := dummy.DeleteDummyModelsFromDB(testDB); err != nil {
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

func matchJSON(a []byte, b []byte, jqExpr *string) {
	var valueA, valueB = a, b
	var err error

	if jqExpr != nil {
		valueA, err = parseJQ(a, *jqExpr)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		valueB, err = parseJQ(b, *jqExpr)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

	}
	Expect(valueA).To(MatchJSON(valueB))
}
