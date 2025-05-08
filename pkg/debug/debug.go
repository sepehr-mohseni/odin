package debug

import (
	"net/http"
	"odin/pkg/config"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func Register(e *echo.Echo, cfg *config.Config, logger *logrus.Logger) {
	e.GET("/debug/routes", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"services": cfg.Services,
		})
	})

	logger.Info("Debug endpoint registered at /debug/routes")
}
