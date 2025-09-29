package rls

import (
	"github.com/flanksource/commons/logger"
	echov4 "github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"

	"github.com/flanksource/duty/context"
)

func Middleware(next echov4.HandlerFunc) echov4.HandlerFunc {
	return func(c echov4.Context) error {
		ctx := c.Request().Context().(context.Context)

		rlsPayload, err := GetPayload(ctx)
		if err != nil {
			return err
		}

		if ctx.Properties().On(false, "rls.debug") {
			ctx.Logger.WithValues("user", lo.FromPtr(ctx.User()).ID).Infof("RLS payload: %s", logger.Pretty(rlsPayload))
		}

		if rlsPayload.Disable {
			return next(c)
		}

		err = ctx.Transaction(func(txCtx context.Context, _ trace.Span) error {
			if err := rlsPayload.SetPostgresSessionRLS(txCtx.DB()); err != nil {
				return err
			}

			txCtx = txCtx.WithRLSPayload(rlsPayload)

			// set the context with the tx
			c.SetRequest(c.Request().WithContext(txCtx))

			return next(c)
		})

		return err
	}
}
