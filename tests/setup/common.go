package setup

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"strings"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var DefaultContext context.Context

var postgresServer *embeddedPG.EmbeddedPostgres
var dummyData dummy.DummyData

var PgUrl string
var postgresDBUrl string
var dbName string

func execPostgres(connection, query string) error {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query)
	return err
}

func MustDB() *sql.DB {
	db, err := duty.NewDB(PgUrl)
	if err != nil {
		panic(err)
	}
	return db
}

func BeforeSuiteFn() context.Context {
	var err error

	if postgresServer != nil {
		return DefaultContext
	}

	port := FreePort()

	PgUrl = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/test?sslmode=disable", port)
	url := os.Getenv("DUTY_DB_URL")
	if url != "" {
		postgresDBUrl = url
		dbName = fmt.Sprintf("duty_gingko%d", port)
		PgUrl = strings.Replace(url, "/postgres", "/"+dbName, 1)
		_ = execPostgres(postgresDBUrl, "DROP DATABASE "+dbName)
		if err := execPostgres(postgresDBUrl, "CREATE DATABASE "+dbName); err != nil {
			panic(fmt.Sprintf("Cannot create %s: %v", dbName, err))
		}
	} else {
		config, _ := GetEmbeddedPGConfig("test", port)
		postgresServer = embeddedPG.NewDatabase(config)
		if err = postgresServer.Start(); err != nil {
			panic(err.Error())
		}
		logger.Infof("Started postgres on port %d", port)
	}

	if ctx, err := duty.InitDB(PgUrl, nil); err != nil {
		panic(err.Error())
	} else {
		DefaultContext = *ctx
	}

	dummyData = dummy.GetStaticDummyData(DefaultContext.DB())
	err = dummyData.Populate(DefaultContext.DB())
	if err != nil {
		panic(err.Error())
	}

	DefaultContext := DefaultContext.WithKubernetes(fake.NewSimpleClientset(&v1.ConfigMap{
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

	logger.StandardLogger().SetLogLevel(2)
	logger.Infof("Created dummy data %v", len(dummyData.Checks))
	return DefaultContext
}

func AfterSuiteFn() {
	// logger.Infof("Deleting dummy data")
	logger.StandardLogger().SetLogLevel(0)
	// testDB, err := duty.NewGorm(PgUrl, duty.DefaultGormConfig())
	// if err != nil {
	// 	ginkgo.Fail(err.Error())
	// }
	// if err := dummyData.Delete(testDB); err != nil {
	// 	ginkgo.Fail(err.Error())
	// }

	if os.Getenv("DUTY_DB_URL") == "" {
		logger.Infof("Stopping postgres")
		if err := postgresServer.Stop(); err != nil {
			ginkgo.Fail(err.Error())
		}
	} else {
		if err := execPostgres(postgresDBUrl, fmt.Sprintf("DROP DATABASE %s (FORCE)", dbName)); err != nil {
			ginkgo.Fail(fmt.Sprintf("Cannot drop %s: %v", dbName, err))
		}
	}
}

func FreePort() int {
	// Bind to port 0 to let the OS choose a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err.Error())
	}

	defer listener.Close()

	// Get the address of the listener
	address := listener.Addr().(*net.TCPAddr)
	return address.Port
}
