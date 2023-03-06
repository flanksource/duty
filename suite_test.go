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
	"github.com/itchyny/gojq"
	"github.com/jackc/pgx/v5/pgxpool"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

func TestDuty(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Duty Suite")
}

var postgresServer *embeddedPG.EmbeddedPostgres

const pgUrl = "postgres://postgres:postgres@localhost:9876/test?sslmode=disable"

var testDB *gorm.DB
var testDBPGPool *pgxpool.Pool

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
	if testDBPGPool, err = NewPgxPool(pgUrl); err != nil {
		ginkgo.Fail(err.Error())
	}
	if _, err := NewDB(pgUrl); err != nil {
		ginkgo.Fail(err.Error())
	}
	err = Migrate(pgUrl)
	Expect(err).ToNot(HaveOccurred())

	testDB, err = NewGorm(pgUrl, DefaultGormConfig())
	Expect(err).ToNot(HaveOccurred())

	err = dummy.PopulateDBWithDummyModels(testDB)
	Expect(err).ToNot(HaveOccurred())
})

var _ = ginkgo.AfterSuite(func() {
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
		Expect(err).ToNot(HaveOccurred())
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
		logger.Infof("JSON VAL %s", string(jsonVal))
	}
	logger.Infof("JSON VAL END ===")

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
	logger.Infof("VAL-A %s", string(valueA))
	logger.Infof("VAL-B %s", string(valueB))
	Expect(valueA).To(MatchJSON(valueB))
}
