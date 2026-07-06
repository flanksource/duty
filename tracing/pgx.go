package tracing

import (
	"context"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
)

// PgxTracer records parameterized SQL shape only and caps SQL attributes.
type PgxTracer struct {
	*otelpgx.Tracer
	formatSQL func(string) string
}

// NewPgxTracer returns a pgx tracer with production-safe defaults.
func NewPgxTracer(opts ...otelpgx.Option) *PgxTracer {
	tracerOpts := []otelpgx.Option{
		otelpgx.WithTrimSQLInSpanName(),
	}
	tracerOpts = append(tracerOpts, opts...)

	return &PgxTracer{
		Tracer:    otelpgx.NewTracer(tracerOpts...),
		formatSQL: SQLStatementTruncator(DefaultMaxSQLStatementLength),
	}
}

func (t *PgxTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	data.SQL = t.formatSQL(data.SQL)
	return t.Tracer.TraceQueryStart(ctx, conn, data)
}

func (t *PgxTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	data.SQL = t.formatSQL(data.SQL)
	t.Tracer.TraceBatchQuery(ctx, conn, data)
}

func (t *PgxTracer) TracePrepareStart(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	data.SQL = t.formatSQL(data.SQL)
	return t.Tracer.TracePrepareStart(ctx, conn, data)
}
