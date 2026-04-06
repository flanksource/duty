package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	embeddedPG "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/context"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/shutdown"
	"github.com/flanksource/duty/telemetry"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type SetupOpts struct {
	DummyData bool
}

type templateInfo struct {
	AdminURL   string `json:"admin_url"`
	TemplateDB string `json:"template_db"`
	Port       int    `json:"port"`
}

func (t templateInfo) Marshal() []byte {
	data, err := json.Marshal(t)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal templateInfo: %v", err))
	}
	return data
}

func unmarshalTemplateInfo(data []byte) templateInfo {
	var info templateInfo
	if err := json.Unmarshal(data, &info); err != nil {
		panic(fmt.Sprintf("failed to unmarshal templateInfo: %v", err))
	}
	return info
}

var (
	adminURL   string
	nodeDBName string
)

func SetupTemplate(opts SetupOpts) []byte {
	if err := properties.LoadFile(findFileInPath("test.properties", 2)); err != nil {
		logger.Errorf("Failed to load test properties: %v", err)
	}

	defer telemetry.InitTracer()

	var port int
	if val, ok := os.LookupEnv(TEST_DB_PORT); ok {
		parsed, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("failed to parse TEST_DB_PORT: %v", err))
		}
		port = int(parsed)
	} else {
		port = duty.FreePort()
	}

	templateDB := "duty_test_template"

	url := os.Getenv(DUTY_DB_URL)
	if url != "" && !recreateDatabase {
		// DUTY_DB_CREATE=false: use direct connection, no template
		PgUrl = url
		return templateInfo{AdminURL: url, TemplateDB: "", Port: port}.Marshal()
	}

	adminConn, err := ensurePostgres(port)
	if err != nil {
		panic(fmt.Sprintf("failed to start postgres: %v", err))
	}
	adminURL = adminConn

	// Always recreate — dummy data uses uuid.New() so a cached template has stale UUIDs
	_ = execPostgres(adminConn, fmt.Sprintf("ALTER DATABASE %s WITH is_template = false", templateDB))
	_ = execPostgres(adminConn, fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()", templateDB))
	_ = execPostgres(adminConn, fmt.Sprintf("DROP DATABASE IF EXISTS %s (FORCE)", templateDB))

	if err := execPostgres(adminConn, fmt.Sprintf("CREATE DATABASE %s", templateDB)); err != nil {
		panic(fmt.Sprintf("failed to create template db: %v", err))
	}

	templateURL := strings.Replace(adminConn, "/postgres", "/"+templateDB, 1)
	if !strings.Contains(adminConn, "/postgres") {
		templateURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, templateDB)
	}

	dbOptions := []duty.StartOption{duty.DisablePostgrest, duty.RunMigrations, duty.WithUrl(templateURL)}
	if !disableRLS {
		dbOptions = append(dbOptions, duty.EnableRLS)
	}

	ctx, stop, err := duty.Start(templateDB, dbOptions...)
	if err != nil {
		panic(fmt.Sprintf("failed to start duty for template: %v", err))
	}

	if err := ctx.DB().Exec("SET TIME ZONE 'UTC'").Error; err != nil {
		panic(fmt.Sprintf("failed to set timezone: %v", err))
	}

	if opts.DummyData {
		dummyData = dummy.GetStaticDummyData(ctx.DB())
		if err := dummyData.Delete(ctx.DB()); err != nil {
			logger.Errorf(err.Error())
		}
		if err := dummyData.Populate(ctx); err != nil {
			panic(fmt.Sprintf("failed to populate dummy data: %v", err))
		}
		logger.Infof("Created dummy data in template (%d checks)", len(dummyData.Checks))
	}

	// Close all connections so the DB can be used as a template
	stop()
	_ = execPostgres(adminConn, fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()", templateDB))

	if err := execPostgres(adminConn, fmt.Sprintf("ALTER DATABASE %s WITH is_template = true", templateDB)); err != nil {
		panic(fmt.Sprintf("failed to mark template db: %v", err))
	}

	return templateInfo{AdminURL: adminConn, TemplateDB: templateDB, Port: port}.Marshal()
}

func SetupNode(data []byte, opts SetupOpts) context.Context {
	info := unmarshalTemplateInfo(data)

	if info.TemplateDB == "" {
		// Direct connection mode (DUTY_DB_CREATE=false)
		PgUrl = info.AdminURL
		ctx, _, err := duty.Start("direct", duty.ClientOnly, duty.WithUrl(PgUrl))
		if err != nil {
			panic(fmt.Sprintf("failed to connect to db: %v", err))
		}
		return setupNodeContext(ctx, "direct")
	}

	adminURL = info.AdminURL
	nodeDBName = fmt.Sprintf("duty_test_node%d", ginkgo.GinkgoParallelProcess())

	// Drop and clone from template
	_ = execPostgres(adminURL, fmt.Sprintf("DROP DATABASE IF EXISTS %s (FORCE)", nodeDBName))

	// Terminate any lingering connections to the template before cloning
	_ = execPostgres(adminURL, fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()", info.TemplateDB))

	// Unmark template temporarily for cloning (some pg versions need this)
	_ = execPostgres(adminURL, fmt.Sprintf("ALTER DATABASE %s WITH is_template = false", info.TemplateDB))
	if err := execPostgres(adminURL, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", nodeDBName, info.TemplateDB)); err != nil {
		panic(fmt.Sprintf("failed to clone template: %v", err))
	}
	_ = execPostgres(adminURL, fmt.Sprintf("ALTER DATABASE %s WITH is_template = true", info.TemplateDB))

	// Build node connection URL
	if strings.Contains(adminURL, "/postgres") {
		PgUrl = strings.Replace(adminURL, "/postgres", "/"+nodeDBName, 1)
	} else {
		PgUrl = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", info.Port, nodeDBName)
	}

	// Skip migrations — the clone is byte-for-byte identical to the template
	ctx, _, err := duty.Start(nodeDBName, duty.ClientOnly, duty.WithUrl(PgUrl))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to node db: %v", err))
	}

	return setupNodeContext(ctx, nodeDBName)
}

func setupNodeContext(ctx context.Context, dbName string) context.Context {
	if err := ctx.DB().Exec("SET TIME ZONE 'UTC'").Error; err != nil {
		panic(fmt.Sprintf("failed to set timezone: %v", err))
	}

	ctx = ctx.WithValue("db_name", dbName).WithValue("db_url", PgUrl)

	clientset := fake.NewClientset(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "default"},
		Data:       map[string]string{"foo": "bar"},
	}, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
		Data:       map[string][]byte{"foo": []byte("secret")},
	})

	return ctx.WithLocalKubernetes(dutyKubernetes.NewKubeClient(logger.GetLogger("k8s"), clientset, nil))
}

func SynchronizedAfterSuiteAllNodes() {
	if nodeDBName != "" && adminURL != "" {
		if err := execPostgres(adminURL, fmt.Sprintf("DROP DATABASE IF EXISTS %s (FORCE)", nodeDBName)); err != nil {
			logger.Errorf("failed to drop node db: %v", err)
		}
	}
}

func SynchronizedAfterSuiteNode1() {
	shutdown.Shutdown()
}


func ensurePostgres(port int) (string, error) {
	url := os.Getenv(DUTY_DB_URL)
	if url != "" {
		postgresDBUrl = url
		return url, nil
	}

	if postgresServer == nil {
		config, _ := GetEmbeddedPGConfig("postgres", port)

		if v, ok := os.LookupEnv(DUTY_DB_DATA_DIR); ok {
			config = config.DataPath(v)
		}

		postgresServer = embeddedPG.NewDatabase(config)
		logger.Infof("starting embedded postgres on port %d", port)
		if err := postgresServer.Start(); err != nil {
			return "", err
		}
		shutdown.AddHookWithPriority("stop embedded postgres", shutdown.PriorityCritical, func() {
			if err := postgresServer.Stop(); err != nil {
				logger.Errorf("failed to stop embedded postgres: %v", err)
			}
		})
		logger.Infof("Started postgres on port %d", port)
	}

	return fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port), nil
}
