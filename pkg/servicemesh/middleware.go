package servicemesh

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Middleware creates an Echo middleware that injects service mesh headers
func Middleware(manager *Manager, logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if mesh is not enabled
			if manager == nil || manager.GetMeshType() == MeshTypeNone {
				return next(c)
			}

			// Inject mesh-specific headers into the request
			if err := manager.InjectHeaders(c.Request().Header); err != nil {
				logger.WithError(err).Warn("Failed to inject mesh headers")
			}

			// Continue with the request
			return next(c)
		}
	}
}

// ProxyMiddleware creates middleware for proxying requests through the mesh
func ProxyMiddleware(manager *Manager, logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Add mesh type to context for downstream handlers
			if manager != nil {
				c.Set("mesh_type", manager.GetMeshType())
				c.Set("mesh_manager", manager)
			}

			return next(c)
		}
	}
}
