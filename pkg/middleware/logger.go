package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func LoggerMiddleware(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()

			err := next(c)

			res := c.Response()
			latency := time.Since(start)

			fields := logrus.Fields{
				"method":     req.Method,
				"uri":        req.RequestURI,
				"status":     res.Status,
				"latency_ms": latency.Milliseconds(),
				"user_agent": req.UserAgent(),
				"ip":         c.RealIP(),
			}

			if err != nil {
				fields["error"] = err.Error()
				logger.WithFields(fields).Error("Request error")
			} else {
				logger.WithFields(fields).Info("Request processed")
			}

			return err
		}
	}
}
