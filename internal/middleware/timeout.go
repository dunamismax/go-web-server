package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v4"
)

// RequestTimeout adds a deadline to the request context without swapping the response writer.
// This avoids the incompatibilities in Echo's Timeout middleware for streaming or HTML-oriented responses.
func RequestTimeout(timeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if timeout <= 0 {
				return next(c)
			}

			ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
			defer cancel()

			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				if err == nil {
					err = ctx.Err()
				}

				return ErrTimeout.WithContext(c).WithInternal(err)
			}

			return err
		}
	}
}
