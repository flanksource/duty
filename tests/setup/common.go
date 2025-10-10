package setup

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/rodaine/table"
	"github.com/samber/oops"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	//fix indirect go.mod
	_ "github.com/spf13/cobra"

	"github.com/flanksource/duty"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac"
	"github.com/flanksource/duty/shutdown"
	"github.com/flanksource/duty/telemetry"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

const (
	DUTY_DB_CREATE      = "DUTY_DB_CREATE"
	DUTY_DB_DISABLE_RLS = "DUTY_DB_DISABLE_RLS"
)

// Env vars for embedded db
const (
	DUTY_DB_DATA_DIR = "DUTY_DB_DATA_DIR"
	DUTY_DB_URL      = "DUTY_DB_URL"
	TEST_DB_PORT     = "TEST_DB_PORT"
)

var DefaultContext context.Context

var postgresServer *embeddedPG.EmbeddedPostgres
var dummyData dummy.DummyData

var PgUrl string
var postgresDBUrl string

func RestartEmbeddedPG() error {
	if err := postgresServer.Stop(); err != nil {
		return err
	}

	return postgresServer.Start()
}

func init() {
	logger.UseSlog()
	logger.BindFlags(pflag.CommandLine)
	duty.BindPFlags(pflag.CommandLine)
	properties.BindFlags(pflag.CommandLine)

	format.RegisterCustomFormatter(func(value interface{}) (string, bool) {
		switch v := value.(type) {
		case error:
			if err, ok := oops.AsOops(v); ok {
				return fmt.Sprintf("%+v", err), true
			}
		}
		return "", false
	})
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

var WithoutRLS = "rls_disabled"
var WithoutDummyData = "without_dummy_data"
var WithExistingDatabase = "with_existing_database"

var (
	recreateDatabase = os.Getenv(DUTY_DB_CREATE) != "false"
	disableRLS       = os.Getenv(DUTY_DB_DISABLE_RLS) == "true"
)

func findFileInPath(filename string, depth int) string {
	if !path.IsAbs(filename) {
		cwd, _ := os.Getwd()
		filename = path.Join(cwd, filename)
	}

	base := path.Base(filename)

	paths := strings.Split(path.Dir(filename), "/")
	for i := len(paths); i >= len(paths)-depth; i-- {
		file := "/" + path.Join(append(paths[:i], base)...)
		logger.Infof(file)
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	return filename
}

func BeforeSuiteFn(args ...interface{}) context.Context {
	ctx, err := SetupDB("test", args...)
	if err != nil {
		shutdown.Shutdown()
		Expect(err).To(BeNil())
	}

	DefaultContext = ctx
	return ctx
}

func SetupDB(dbName string, args ...interface{}) (context.Context, error) {
	if err := properties.LoadFile(findFileInPath("test.properties", 2)); err != nil {
		logger.Errorf("Failed to load test properties: %v", err)
	}

	defer telemetry.InitTracer()

	importDummyData := true
	dbOptions := []duty.StartOption{duty.DisablePostgrest, duty.RunMigrations}
	for _, arg := range args {
		if arg == WithoutDummyData {
			importDummyData = false
		}
		if arg == WithExistingDatabase {
			recreateDatabase = false
		}
		if arg == WithoutRLS {
			disableRLS = true
		}
	}

	if !disableRLS {
		dbOptions = append(dbOptions, duty.EnableRLS)
	}

	var port int
	if val, ok := os.LookupEnv(TEST_DB_PORT); ok {
		parsed, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return context.Context{}, err
		}

		port = int(parsed)
	} else {
		port = duty.FreePort()
	}

	PgUrl = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, dbName)
	url := os.Getenv(DUTY_DB_URL)
	if url != "" && !recreateDatabase {
		PgUrl = url
	} else if url != "" && recreateDatabase {
		postgresDBUrl = url
		dbName = fmt.Sprintf("duty_gingko%d", port)
		PgUrl = strings.Replace(url, "/postgres", "/"+dbName, 1)
		_ = execPostgres(postgresDBUrl, "DROP DATABASE "+dbName)
		if err := execPostgres(postgresDBUrl, "CREATE DATABASE "+dbName); err != nil {
			return context.Context{}, fmt.Errorf("cannot create %s: %v", dbName, err)
		}

		shutdown.AddHookWithPriority("remote postgres", shutdown.PriorityCritical, func() {
			if err := execPostgres(postgresDBUrl, fmt.Sprintf("DROP DATABASE %s (FORCE)", dbName)); err != nil {
				logger.Errorf("execPostgres: %v", err)
			}
		})

	} else if url == "" && postgresServer == nil {
		config, _ := GetEmbeddedPGConfig(dbName, port)

		// allow data dir override
		if v, ok := os.LookupEnv(DUTY_DB_DATA_DIR); ok {
			config = config.DataPath(v)
		}

		postgresServer = embeddedPG.NewDatabase(config)
		logger.Infof("starting embedded postgres on port %d", port)
		if err := postgresServer.Start(); err != nil {
			return context.Context{}, err
		}
		logger.Infof("Started postgres on port %d", port)
		shutdown.AddHookWithPriority("embedded pg", shutdown.PriorityCritical, func() {
			if err := postgresServer.Stop(); err != nil {
				logger.Errorf("postgresServer.Stop: %v", err)
			}
		})
	}

	dbOptions = append(dbOptions, duty.WithUrl(PgUrl))
	ctx, _, err := duty.Start(dbName, dbOptions...)
	if err != nil {
		return context.Context{}, err
	}

	if err := ctx.DB().Exec("SET TIME ZONE 'UTC'").Error; err != nil {
		return context.Context{}, err
	}

	ctx = ctx.WithValue("db_name", dbName).WithValue("db_url", PgUrl)

	if importDummyData {
		dummyData = dummy.GetStaticDummyData(ctx.DB())
		if err := dummyData.Delete(ctx.DB()); err != nil {
			logger.Errorf(err.Error())
		}
		err = dummyData.Populate(ctx)
		if err != nil {
			return context.Context{}, err
		}
		logger.Infof("Created dummy data %v", len(dummyData.Checks))
	}

	clientset := fake.NewSimpleClientset(&v1.ConfigMap{
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

	ctx = ctx.WithLocalKubernetes(dutyKubernetes.NewKubeClient(logger.GetLogger("k8s"), clientset, nil))

	return ctx, nil
}

func AfterSuiteFn() {
	shutdown.Shutdown()
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

	config := api.NewConfig(strings.ReplaceAll(pgUrl, pgDbName, newName))

	dbConfig := duty.RunMigrations(config)
	if !disableRLS {
		dbConfig = duty.EnableRLS(dbConfig)
	}

	newCtx, err := duty.InitDB(dbConfig)
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
	port := duty.FreePort()

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

func ExpectJobToPass(j *job.Job) {
	history, err := j.FindHistory()
	Expect(err).To(BeNil())
	Expect(len(history)).To(BeNumerically(">=", 1))
	Expect(history[0].Status).To(BeElementOf(models.StatusSuccess))
}

func DumpEventQueue(ctx context.Context) {
	var events []models.Event
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

// CreateUserWithRole creates a user and assigns the specified roles
func CreateUserWithRole(ctx context.Context, name, email string, roles ...string) *models.Person {
	user := &models.Person{
		Name:  name,
		Email: email,
	}
	err := ctx.DB().Create(user).Error
	Expect(err).ToNot(HaveOccurred())

	for _, role := range roles {
		_, err = rbac.Enforcer().AddRoleForUser(user.ID.String(), role)
		Expect(err).ToNot(HaveOccurred())
	}

	return user
}
