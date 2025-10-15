package health

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Status struct {
	Status    string               `json:"status"`
	Timestamp int64                `json:"timestamp"`
	Checks    map[string]CheckInfo `json:"checks,omitempty"`
}

type CheckInfo struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type CheckFunc func() error

type Checker struct {
	checks map[string]CheckFunc
}

func NewChecker() *Checker {
	return &Checker{
		checks: make(map[string]CheckFunc),
	}
}

func (c *Checker) AddCheck(name string, check CheckFunc) {
	c.checks[name] = check
}

func (c *Checker) Check() *Status {
	status := &Status{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Checks:    make(map[string]CheckInfo),
	}

	for name, check := range c.checks {
		checkInfo := CheckInfo{
			Timestamp: time.Now().Unix(),
		}

		if err := check(); err != nil {
			checkInfo.Status = "unhealthy"
			checkInfo.Error = err.Error()
			status.Status = "unhealthy"
		} else {
			checkInfo.Status = "healthy"
		}

		status.Checks[name] = checkInfo
	}

	return status
}

func (c *Checker) Readiness() *Status {
	return &Status{
		Status:    "ready",
		Timestamp: time.Now().Unix(),
	}
}

func (c *Checker) Liveness() *Status {
	return &Status{
		Status:    "alive",
		Timestamp: time.Now().Unix(),
	}
}

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
		var routes []map[string]interface{}
		for _, route := range e.Routes() {
			routes = append(routes, map[string]interface{}{
				"method": route.Method,
				"path":   route.Path,
				"name":   route.Name,
			})
		}
		return c.JSON(http.StatusOK, routes)
	})

	e.GET("/debug/config", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	e.GET("/debug/content-types", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"supported": []string{
				"application/json",
				"text/html",
				"text/plain",
			},
		})
	})

	logger.Info("Health check and debug endpoints registered")
}
