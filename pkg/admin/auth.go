package admin

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func (h *AdminHandler) basicAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !h.enabled {
			return next(c)
		}

		auth := c.Request().Header.Get("Authorization")

		if auth == "" {
			cookie, err := c.Cookie("Authorization")
			if err == nil && cookie.Value != "" {
				auth = cookie.Value
				if !strings.HasPrefix(auth, "Basic ") {
					auth = "Basic " + auth
				}
			}
		}

		if auth == "" {
			return h.unauthorized(c)
		}

		const basicAuthPrefix = "Basic "
		if !strings.HasPrefix(auth, basicAuthPrefix) {
			return h.unauthorized(c)
		}

		payload, err := base64.StdEncoding.DecodeString(auth[len(basicAuthPrefix):])
		if err != nil {
			return h.unauthorized(c)
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 || pair[0] != h.username || pair[1] != h.password {
			return h.unauthorized(c)
		}

		return next(c)
	}
}

func (h *AdminHandler) unauthorized(c echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Basic realm="Admin Area"`)
	return c.HTML(http.StatusUnauthorized, `
		<html>
		<head><title>Authentication Required</title></head>
		<body>
			<h1>Authentication Required</h1>
			<p>Please login with admin credentials.</p>
			<script>
				window.location.href = "/admin/login";
			</script>
		</body>
		</html>
	`)
}

func (h *AdminHandler) handleLogin(c echo.Context) error {
	return h.renderTemplate(c, "login.html", nil)
}

func (h *AdminHandler) handleLoginPost(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == h.username && password == h.password {
		authValue := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		cookie := http.Cookie{
			Name:     "Authorization",
			Value:    authValue,
			Path:     "/admin",
			HttpOnly: true,
			MaxAge:   3600 * 24,
		}
		c.SetCookie(&cookie)

		c.Response().Header().Set("HX-Redirect", "/admin/dashboard")
		return c.String(http.StatusOK, "Login successful. Redirecting...")
	}

	return c.HTML(http.StatusUnauthorized, `<div class="alert alert-danger">Invalid username or password</div>`)
}
