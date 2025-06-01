package health

import (
	"net/http"
	"odin/pkg/config"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func Register(e *echo.Echo, logger *logrus.Logger) {
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "UP",
		})
	})

	// Add readiness endpoint
	e.GET("/ready", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ready",
		})
	})

	// Add liveness endpoint
	e.GET("/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "alive",
		})
	})

	e.GET("/debug/routes", func(c echo.Context) error {
		routes := []map[string]string{}
		for _, r := range e.Routes() {
			routes = append(routes, map[string]string{
				"method": r.Method,
				"path":   r.Path,
			})
		}
		return c.JSON(http.StatusOK, routes)
	})

	e.GET("/debug/config", func(c echo.Context) error {
		if cfg, ok := c.Get("config").(*config.Config); ok {
			return c.JSON(http.StatusOK, cfg)
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status": "Config not available in context",
		})
	})

	e.GET("/debug/content-types", func(c echo.Context) error {
		testObj := map[string]interface{}{
			"message":   "This is a JSON test message",
			"timestamp": time.Now().Unix(),
		}

		c.Response().Header().Set("Content-Type", "application/json")
		return c.JSON(http.StatusOK, testObj)
	})

	logger.Info("Health check and debug endpoints registered")
}
