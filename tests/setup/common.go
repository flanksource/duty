package setup

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/postq"
	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rodaine/table"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var DefaultContext context.Context

var postgresServer *embeddedPG.EmbeddedPostgres
var dummyData dummy.DummyData

var PgUrl string
var postgresDBUrl string
var dbName = "test"
var trace bool
var dbTrace bool

func init() {
	logger.BindGoFlags()
	duty.BindGoFlags()
}

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

var WithoutDummyData = "without_dummy_data"
var WithExistingDatabase = "with_existing_database"
var recreateDatabase = os.Getenv("DUTY_DB_CREATE") != "false"

func BeforeSuiteFn(args ...interface{}) context.Context {
	logger.UseZap()
	var err error
	importDummyData := true

	for _, arg := range args {
		if arg == WithoutDummyData {
			importDummyData = false
		}
		if arg == WithExistingDatabase {
			recreateDatabase = false
		}
	}

	logger.Infof("Initializing test db debug=%v db.trace=%v", trace, dbTrace)
	if postgresServer != nil {
		return DefaultContext
	}

	var port int
	if val, ok := os.LookupEnv("TEST_DB_PORT"); ok {
		parsed, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			panic(err)
		}

		port = int(parsed)
	} else {
		port = FreePort()
	}

	PgUrl = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, dbName)
	url := os.Getenv("DUTY_DB_URL")
	if url != "" && !recreateDatabase {
		PgUrl = url
	} else if url != "" && recreateDatabase {
		postgresDBUrl = url
		dbName = fmt.Sprintf("duty_gingko%d", port)
		PgUrl = strings.Replace(url, "/postgres", "/"+dbName, 1)
		_ = execPostgres(postgresDBUrl, "DROP DATABASE "+dbName)
		if err := execPostgres(postgresDBUrl, "CREATE DATABASE "+dbName); err != nil {
			panic(fmt.Sprintf("Cannot create %s: %v", dbName, err))
		}
	} else if url == "" {
		config, _ := GetEmbeddedPGConfig(dbName, port)
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

	if err := DefaultContext.DB().Exec("SET TIME ZONE 'UTC'").Error; err != nil {
		panic(err.Error())
	}

	DefaultContext = context.Context{
		Context: DefaultContext.WithValue("db_name", dbName).WithValue("db_url", PgUrl),
	}

	if importDummyData {
		dummyData = dummy.GetStaticDummyData(DefaultContext.DB())
		dummyData.Delete(DefaultContext.DB())
		err = dummyData.Populate(DefaultContext.DB())
		if err != nil {
			panic(err.Error())
		}
		logger.Infof("Created dummy data %v", len(dummyData.Checks))
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

	if dbTrace {
		DefaultContext = DefaultContext.WithDBLogLevel("trace")
	}
	if trace {
		DefaultContext = DefaultContext.WithTrace()
	}
	return DefaultContext
}

func AfterSuiteFn() {
	if os.Getenv("DUTY_DB_URL") == "" {
		logger.Infof("Stopping postgres")
		if err := postgresServer.Stop(); err != nil {
			ginkgo.Fail(err.Error())
		}
	} else if recreateDatabase {
		if err := execPostgres(postgresDBUrl, fmt.Sprintf("DROP DATABASE %s (FORCE)", dbName)); err != nil {
			ginkgo.Fail(fmt.Sprintf("Cannot drop %s: %v", dbName, err))
		}
	}
}

// NewDB creates a new database from an existing context, and
// returns the new context and a function that be called to drop it
func NewDB(ctx context.Context, name string) (*context.Context, func(), error) {
	pgUrl := ctx.Value("db_url").(string)
	pgDbName := ctx.Value("db_name").(string)
	newName := pgDbName + name

	if err := ctx.DB().Exec(fmt.Sprintf("CREATE DATABASE %s", newName)).Error; err != nil {
		return nil, nil, err
	}

	newCtx, err := duty.InitDB(strings.ReplaceAll(pgUrl, pgDbName, newName), nil)
	if err != nil {
		return nil, nil, err
	}

	if err := newCtx.DB().Exec("SET TIME ZONE 'UTC'").Error; err != nil {
		return nil, nil, err
	}

	return newCtx, func() {
		if err := ctx.DB().Exec(fmt.Sprintf("DROP DATABASE  %s (FORCE)", newName)).Error; err != nil {
			logger.Errorf("error cleaning up db: %v", err)
		}
	}, nil
}

func RunEcho(e *echo.Echo) (int, func()) {
	port := FreePort()

	listenAddr := fmt.Sprintf(":%d", port)
	go func() {
		defer ginkgo.GinkgoRecover() // Required by ginkgo, if an assertion is made in a goroutine.
		if err := e.Start(listenAddr); err != nil {
			if err == http.ErrServerClosed {
				logger.Infof("Server closed")
			} else {
				ginkgo.Fail(fmt.Sprintf("Failed to start test server: %v", err))
			}
		}
	}()
	return port, func() {
		defer ginkgo.GinkgoRecover() // Required by ginkgo, if an assertion is made in a goroutine.
		Expect(e.Close()).To(BeNil())
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

func ExpectJobToPass(j *job.Job) {
	history, err := j.FindHistory()
	Expect(err).To(BeNil())
	Expect(len(history)).To(BeNumerically(">=", 1))
	Expect(history[0].Status).To(BeElementOf(models.StatusSuccess))
}

func DumpEventQueue(ctx context.Context) {
	var events []postq.Event
	Expect(ctx.DB().Find(&events).Error).To(BeNil())

	table.DefaultHeaderFormatter = func(format string, vals ...interface{}) string {
		return strings.ToUpper(fmt.Sprintf(format, vals...))
	}

	tbl := table.New("Event", "Created At", "Properties")

	for _, event := range events {
		tbl.AddRow(event.Name, event.CreatedAt, event.Properties)
	}

	tbl.Print()
}
