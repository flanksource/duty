package setup

import (
	"database/sql"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/setup"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/testutils"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var postgresServer *embeddedPG.EmbeddedPostgres
var dummyData dummy.DummyData

const PgUrl = "postgres://postgres:postgres@localhost:9876/test?sslmode=disable"

func MustDB() *sql.DB {
	db, err := setup.NewDB(PgUrl)
	if err != nil {
		panic(err)
	}
	return db
}

func BeforeSuiteFn() {
	var err error

	if postgresServer != nil {
		return
	}
	config, _ := testutils.GetEmbeddedPGConfig("test", 9876)
	postgresServer = embeddedPG.NewDatabase(config)
	if err = postgresServer.Start(); err != nil {
		ginkgo.Fail(err.Error())
	}
	logger.Infof("Started postgres on port 9876")

	if ctx, err := setup.InitDB(PgUrl, nil); err != nil {
		ginkgo.Fail(err.Error())
	} else {
		testutils.DefaultContext = *ctx
	}

	dummyData = dummy.GetStaticDummyData(testutils.DefaultContext.DB())
	err = dummyData.Populate(testutils.DefaultContext.DB())
	Expect(err).ToNot(HaveOccurred())

	ctx := testutils.DefaultContext.WithKubernetes(fake.NewSimpleClientset(&v1.ConfigMap{
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
		}}))

	testutils.DefaultContext = ctx
}

func AfterSuiteFn() {
	logger.Infof("Deleting dummy data")
	testDB, err := setup.NewGorm(PgUrl, setup.DefaultGormConfig())
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
}
