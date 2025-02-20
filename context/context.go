package context

import (
	gocontext "context"
	"fmt"
	"reflect"
	"strings"
	"time"

	commons "github.com/flanksource/commons/context"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/cache"
	dutyGorm "github.com/flanksource/duty/gorm"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/tracing"
	"github.com/flanksource/duty/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ContextKey string

const rlsPayloadCtxKey ContextKey = "rls-payload"

func init() {
	logger.SkipFrameSuffixes = append(logger.SkipFrameSuffixes, "context/context.go")
}

type Context struct {
	commons.Context
}

func (k Context) Oops(tags ...string) oops.OopsErrorBuilder {
	var args []any

	for k, v := range k.GetLoggingContext() {
		args = append(args, k, v)
	}
	return oops.With(args...).Tags(tags...)
}

func New(opts ...commons.ContextOptions) Context {
	return NewContext(gocontext.Background(), opts...)
}

func NewContext(baseCtx gocontext.Context, opts ...commons.ContextOptions) Context {
	baseOpts := []commons.ContextOptions{
		commons.WithDebugFn(func(ctx commons.Context) *bool {
			for _, o := range Objects(ctx) {
				annotations := getObjectMeta(o).Annotations
				if annotations != nil && (annotations["debug"] == "true" || annotations["trace"] == "true") {
					return lo.ToPtr(true)
				}
			}
			return nil
		}),
		commons.WithTraceFn(func(ctx commons.Context) *bool {
			for _, o := range Objects(ctx) {
				annotations := getObjectMeta(o).Annotations
				if annotations != nil && annotations["trace"] == "true" {
					return lo.ToPtr(true)
				}
			}
			return nil
		}),
	}
	baseOpts = append(baseOpts, opts...)
	ctx := commons.NewContext(
		baseCtx,
		baseOpts...,
	)
	if ctx.Logger == nil {
		ctx.Logger = logger.StandardLogger()
	}
	return Context{
		Context: ctx,
	}
}

func (k Context) String() string {
	s := []string{}
	if k.IsTrace() {
		s = append(s, "[trace]")
	} else if k.IsDebug() {
		s = append(s, "[debug]")
	}

	s = append(s, fmt.Sprintf("logger={%s}", k.Context.String()))

	if user := k.User(); user != nil {
		s = append(s, fmt.Sprintf("user=%s", user.Name))
	}

	if ns := k.GetNamespace(); ns != "" {
		s = append(s, fmt.Sprintf("namespace=%s", ns))
	}
	if name := k.GetName(); name != "" {
		s = append(s, fmt.Sprintf("name=%s", name))
	}
	return strings.Join(s, " ")
}

func (k Context) WithTimeout(timeout time.Duration) (Context, gocontext.CancelFunc) {
	ctx, cancelFunc := k.Context.WithTimeout(timeout)
	return Context{
		Context: ctx,
	}, cancelFunc
}

func (k Context) WithDeadline(deadline time.Time) (Context, gocontext.CancelFunc) {
	ctx, cancelFunc := k.Context.WithDeadline(deadline)
	return Context{
		Context: ctx,
	}, cancelFunc
}

func (k Context) WithValue(key, val any) Context {
	return Context{
		Context: k.Context.WithValue(key, val),
	}
}

// // Deprecated: use WithValue
func (k Context) WithAnyValue(key, val any) Context {
	return k.WithValue(key, val)
}

func (k Context) WithAppendObject(object any) Context {
	return k.WithObject(append(k.Objects(), object)...)
}

// Order the objects from parent -> child
func (k Context) WithObject(object ...any) Context {
	var logNames []string
	ctx := k

	for _, o := range object {
		switch v := o.(type) {
		case models.NamespaceScopeAccessor:
			ctx = ctx.WithNamespace(v.NamespaceScope())
		}
		switch v := o.(type) {
		case models.LogNameAccessor:
			logNames = append(logNames, v.LoggerName())
		}
	}
	ctx = ctx.WithValue("object", object)
	if len(logNames) > 0 {
		ctx.Logger = logger.GetLogger(strings.Join(logNames, "."))
	}
	return ctx
}

func (k Context) Verbose() logger.Logger {
	var args []any
	for k, v := range k.GetLoggingContext() {
		if lo.IsNotEmpty(v) {
			args = append(args, k, v)
		}
	}

	return k.Logger.WithValues(args...)
}

func (k Context) Objects() []any {
	return Objects(k.Context)
}

func (k Context) WithTopology(topology any) Context {
	return k.WithValue("topology", topology)
}

func (k Context) WithUser(user *models.Person) Context {
	k.GetSpan().SetAttributes(attribute.String("user-id", user.ID.String()))
	return k.WithValue("user", user)
}

// Rbac subject
func (k Context) WithSubject(subject string) Context {
	k.GetSpan().SetAttributes(attribute.String("rbac-subject", subject))
	return k.WithValue("rbac-subject", subject)
}

func (k Context) Subject() string {
	subject := k.Value("rbac-subject")
	if subject != nil {
		return subject.(string)
	}

	user := k.User()
	if user != nil {
		return user.ID.String()
	}

	return ""
}

func (k Context) WithoutName() Context {
	k.Logger = logger.GetLogger()
	return k
}

func (k Context) WithName(name string) Context {
	k.Logger = k.Logger.Named(name)
	return k
}

func (k Context) User() *models.Person {
	v := k.Value("user")
	if v == nil {
		return nil
	}
	return v.(*models.Person)
}

// WithAgent sets the current session's agent in the context
func (k Context) WithAgent(agent models.Agent) Context {
	k.GetSpan().SetAttributes(attribute.String("agent-id", agent.ID.String()))
	return k.WithValue("agent", agent)
}

func (k Context) Agent() *models.Agent {
	v := k.Value("agent")
	if v == nil {
		return nil
	}
	return lo.ToPtr(v.(models.Agent))
}

func (k Context) WithTrace() Context {
	return Context{
		Context: k.Context.WithTrace(),
	}
}

func (k Context) WithDebug() Context {
	return Context{
		Context: k.Context.WithDebug(),
	}
}

type KubernetesConnection interface {
	Populate(Context, bool) (kubernetes.Interface, *rest.Config, error)
	Hash() string
	CanExpire() bool
}

func (k Context) WithKubernetes(conn KubernetesConnection) Context {
	if conn == nil {
		return k
	}
	return k.WithValue("kubernetes-connection", conn)
}

func (k Context) WithNamespace(namespace string) Context {
	return k.WithValue("namespace", namespace)
}

func (k Context) WithDB(db *gorm.DB, pool *pgxpool.Pool) Context {
	return k.WithValue("db", db).WithValue("pgxpool", pool)
}

// Returns a new named logger, the default db log level starts at INFO for DDL
// and then increases to TRACE1 depending on the query type and rows returned
// set a baseLevel at Debug, will increase all the levels by 1
func (k Context) WithDBLogger(name string, baseLevel any) Context {
	db := k.DB().Session(&gorm.Session{
		Context: k.Context,
	})
	db.Logger = db.Logger.(*dutyGorm.SqlLogger).WithLogger(name, baseLevel)
	return k.WithValue("db", db)
}

// Changes the minimum log level for db statements
func (k Context) WithDBLogLevel(level any) Context {
	db := k.DB().Session(&gorm.Session{
		Context: k.Context,
	})
	db.Logger = db.Logger.(*dutyGorm.SqlLogger).WithLogLevel(level)
	return k.WithValue("db", db)
}

// FastDB returns a db suitable for high-performance usage, with limited logging and tracing
func (k Context) FastDB(name ...string) *gorm.DB {
	return k.Fast(name...).DB()
}

// Fast with limiting tracing and db logging
func (k Context) Fast(name ...string) Context {
	if len(name) > 0 {
		return k.WithoutTracing().WithDBLogger(name[0], logger.Trace)
	}
	return k.WithoutTracing().WithDBLogger("db", logger.Trace)
}

func (k Context) IsTracing() bool {
	return k.Value(tracing.TracePaused) == nil
}

func (k Context) WithoutTracing() Context {
	return k.WithValue(tracing.TracePaused, "true")
}

func (k Context) Transaction(fn func(ctx Context, span trace.Span) error, opts ...any) error {
	return k.DB().Transaction(func(tx *gorm.DB) error {
		ctx := k.WithDB(tx, k.Pool())
		for _, opt := range opts {
			switch v := opt.(type) {
			case string:
				ctx, span := ctx.StartSpan(v)
				defer span.End()
				return fn(ctx, span)
			}
		}
		return fn(ctx, noop.Span{})
	})
}

func (k Context) DB() *gorm.DB {
	val := k.Value("db")
	if val == nil {
		return nil
	}

	v, ok := val.(*gorm.DB)
	if !ok || v == nil {
		return nil
	}
	return v.WithContext(k)
}

func (k Context) Pool() *pgxpool.Pool {
	val := k.Value("pgxpool")
	if val == nil {
		return nil
	}
	v, ok := val.(*pgxpool.Pool)
	if !ok || v == nil {
		return nil
	}
	return v

}

// KubeAuthFingerprint generates a unique SHA-256 hash to identify the Kubernetes API server
// and client authentication details from the REST configuration.
func (k *Context) KubeAuthFingerprint() string {
	kc, _ := k.Kubernetes()
	if kc == nil {
		return ""
	}
	rc := kc.RestConfig()
	if rc == nil {
		return ""
	}
	return dutyKubernetes.RestConfigFingerprint(rc)
}

func (k *Context) KubernetesConnection() KubernetesConnection {
	if v, ok := k.Value("kubernetes-connection").(KubernetesConnection); ok {
		return v
	}
	return nil
}

type KubernetesClient struct {
	*dutyKubernetes.Client
	Connection KubernetesConnection
	expiry     time.Time
}

func (c *KubernetesClient) SetExpiry(d time.Duration) {
	c.expiry = time.Now().Add(d)
}

func (c *KubernetesClient) RefreshWithExpiry(ctx Context, d time.Duration) error {
	if !c.HasExpired() {
		return nil
	}
	_, rc, err := c.Connection.Populate(ctx, true)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	// Update rest config in place for easy reuse
	c.Config.Host = rc.Host
	c.Config.TLSClientConfig = rc.TLSClientConfig
	c.Config.BearerToken = rc.BearerToken

	c.SetExpiry(15 * time.Minute)
	return nil
}

func (c KubernetesClient) HasExpired() bool {
	if c.Connection.CanExpire() {
		return time.Until(c.expiry) <= 0
	}
	return false
}

var k8sclientcache = cache.NewCache[*KubernetesClient]("k8s-client-cache", 24*time.Hour)

func (k Context) Kubernetes() (*dutyKubernetes.Client, error) {
	conn, ok := k.Value("kubernetes-connection").(KubernetesConnection)
	if !ok {
		return nil, fmt.Errorf("invalid type for KubernetesConnection")
	}
	connHash := conn.Hash()
	if client, exists := k8sclientcache.Get(k, connHash); exists == nil {
		client.RefreshWithExpiry(k, 15*time.Minute)
		logger.Infof("From client cache")
		return client.Client, nil
	}
	c, rc, err := conn.Populate(k, true)
	if err != nil {
		return nil, err
	}
	client := &KubernetesClient{
		Client:     dutyKubernetes.NewKubeClient(c, rc),
		Connection: conn,
	}
	client.SetExpiry(15 * time.Minute)
	k8sclientcache.Set(k, connHash, client)
	return client.Client, nil
}

func (k *Context) KubernetesClient() *dutyKubernetes.Client {
	if v, ok := k.Value("kubernetes-client").(*dutyKubernetes.Client); ok {
		return v
	}
	return nil
}

func (k Context) WithRLSPayload(payload *rls.Payload) Context {
	return k.WithValue(rlsPayloadCtxKey, payload)
}

func (k Context) RLSPayload() *rls.Payload {
	v := k.Value(rlsPayloadCtxKey)
	if v == nil {
		return nil
	}

	return v.(*rls.Payload)
}

func (k Context) Topology() any {
	return k.Value("topology")
}

func (k Context) StartSpan(name string) (Context, trace.Span) {
	ctx, span := k.Context.StartSpan(name)
	for k, v := range k.GetLoggingContext() {
		span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
	}
	return k.Wrap(ctx).WithName(name), span
}

func (k Context) WrapEcho(c echo.Context) Context {
	c2 := k.Wrap(c.Request().Context())
	if c.Request().Header.Get("X-Trace") == "true" {
		c2 = c2.WithTrace()
	}
	if c.Request().Header.Get("X-Debug") == "true" {
		c2 = c2.WithDebug()
	}
	return c2
}

func (k Context) GetLoggingContext() map[string]any {
	meta := k.GetObjectMeta()
	args := map[string]any{
		"namespace": meta.Namespace,
		"name":      meta.Name,
	}

	if user := k.User(); user != nil {
		args["user"] = user.Name
	}
	if agent := k.Agent(); agent != nil {
		args["agent"] = agent.ID
	}

	for _, o := range k.Objects() {
		switch v := o.(type) {
		case ContextAccessor:
			for k, v := range v.Context() {
				if lo.IsNotEmpty(v) {
					args[k] = v
				}
			}

		case ContextAccessor2:
			for k, v := range v.GetContext() {
				if lo.IsNotEmpty(v) {
					args[k] = v
				}
			}
		}
	}

	if m := k.Value("values"); m != nil {
		for k, v := range m.(map[string]interface{}) {
			if !lo.IsEmpty(v) {
				args[k] = v
			}
		}
	}

	return args
}

func (k Context) WithLoggingValues(args ...interface{}) Context {
	var m map[string]interface{}
	if v := k.Value("values"); v != nil {
		m = v.(map[string]interface{})
	} else {
		m = make(map[string]interface{})
	}

	for i := 0; i < len(args)-1; i = i + 2 {
		m[args[i].(string)] = args[i+1]
	}
	for _, arg := range args {
		k = k.WithValue(reflect.TypeOf(arg).Name(), arg)
	}
	return k.WithValue("values", m)
}

func (k Context) GetObjectMeta() metav1.ObjectMeta {
	var meta metav1.ObjectMeta
	for _, o := range k.Objects() {
		meta = getObjectMeta(o)
		if meta.Name != "" && meta.Namespace != "" {
			return meta
		}
	}
	return meta
}

func (k Context) GetNamespace() string {
	if ns := k.GetObjectMeta().Namespace; ns != "" {
		return ns
	}
	if v, ok := k.Value("namespace").(string); ok && v != "" {
		return v
	}
	return ""
}

func (k Context) GetName() string {
	return k.GetObjectMeta().Name
}

func (k Context) GetLabels() map[string]string {
	return k.GetObjectMeta().Labels
}

func (k Context) GetAnnotations() map[string]string {
	return k.GetObjectMeta().Annotations
}

func (k Context) GetEnvValueFromCache(input types.EnvVar, namespace string) (string, error) {
	return GetEnvValueFromCache(k, input, namespace)
}

func (k Context) GetEnvStringFromCache(env string, namespace string) (string, error) {
	return GetEnvStringFromCache(k, env, namespace)
}

func (k Context) GetSecretFromCache(namespace, name, key string) (string, error) {
	return GetSecretFromCache(k, namespace, name, key)
}

func (k Context) GetConfigMapFromCache(namespace, name, key string) (string, error) {
	return GetConfigMapFromCache(k, namespace, name, key)
}

func (k Context) HydrateConnectionByURL(connectionString string) (*models.Connection, error) {
	return HydrateConnectionByURL(k, connectionString)
}

func (k Context) HydrateConnection(connection *models.Connection) (*models.Connection, error) {
	return HydrateConnection(k, connection)
}

func (k Context) Wrap(ctx gocontext.Context) Context {
	return NewContext(ctx, commons.WithTracer(k.GetTracer()), commons.WithLogger(k.Logger)).
		WithDB(k.DB(), k.Pool()).
		WithKubernetes(k.KubernetesConnection()).
		WithNamespace(k.GetNamespace())
}

func stringSliceToMap(s []string) map[string]string {
	m := make(map[string]string)
	for i := 0; i < len(s)-1; i += 2 {
		m[s[i]] = s[i+1]
	}
	return m
}
