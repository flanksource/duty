package types

import (
	"context"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DutyContext interface {
	Oops(tags ...string) oops.OopsErrorBuilder
	String() string
	WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc)
	WithDeadline(deadline time.Time) (context.Context, context.CancelFunc)
	WithValue(key, val any) DutyContext
	WithAnyValue(key, val any) DutyContext
	WithAppendObject(object any) DutyContext
	WithObject(object ...any) DutyContext
	Verbose() logger.Logger
	Objects() []any
	WithTopology(topology any) DutyContext
	WithoutName() DutyContext
	WithName(name string) DutyContext
	WithTrace() DutyContext
	WithDebug() DutyContext
	WithNamespace(namespace string) DutyContext
	WithDB(db *gorm.DB, pool *pgxpool.Pool) DutyContext
	WithDBLogger(name string, baseLevel any) DutyContext
	WithDBLogLevel(level any) DutyContext
	FastDB(name ...string) *gorm.DB
	Fast(name ...string) DutyContext
	IsTracing() bool
	WithoutTracing() DutyContext
	Transaction(fn func(ctx DutyContext, span trace.Span) error, opts ...any) error
	DB() *gorm.DB
	Pool() *pgxpool.Pool
	Kubernetes() kubernetes.Interface
	KubernetesRestConfig() *rest.Config
	StartSpan(name string) (DutyContext, trace.Span)
	GetLoggingDutyContext() map[string]any
	WithLoggingValues(args ...interface{}) DutyContext
	GetNamespace() string
	GetName() string
	GetLabels() map[string]string
	GetAnnotations() map[string]string
	GetEnvStringFromCache(env string, namespace string) (string, error)
	GetSecretFromCache(namespace, name, key string) (string, error)
	GetConfigMapFromCache(namespace, name, key string) (string, error)
	Wrap(ctx context.Context) DutyContext
}
