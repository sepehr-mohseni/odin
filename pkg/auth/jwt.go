package auth

import (
	"fmt"
	"net/http"
	"odin/pkg/config"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

type AuthSecrets struct {
	JWTSecret string `yaml:"jwtSecret"`
}

type JWTConfig struct {
	Secret          string        `yaml:"secret"`
	AccessTokenTTL  time.Duration `yaml:"accessTokenTTL"`
	RefreshTokenTTL time.Duration `yaml:"refreshTokenTTL"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTManager(config JWTConfig) *JWTManager {
	return &JWTManager{
		secret:          config.Secret,
		accessTokenTTL:  config.AccessTokenTTL,
		refreshTokenTTL: config.RefreshTokenTTL,
	}
}

func (jm *JWTManager) GenerateToken(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jm.secret))
}

func (jm *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jm.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func loadJWTSecret() (string, error) {
	secretsPaths := []string{
		"config/auth_secrets.yaml",
		"/etc/odin/auth_secrets.yaml",
		filepath.Join(os.Getenv("HOME"), ".odin", "auth_secrets.yaml"),
	}

	for _, path := range secretsPaths {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var secrets AuthSecrets
			if err := yaml.Unmarshal(data, &secrets); err != nil {
				continue
			}

			if secrets.JWTSecret != "" {
				return secrets.JWTSecret, nil
			}
		}
	}

	if envSecret := os.Getenv("ODIN_JWT_SECRET"); envSecret != "" {
		return envSecret, nil
	}

	return "", fmt.Errorf("couldn't load JWT secret from any source")
}

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTMiddleware(config config.AuthConfig) echo.MiddlewareFunc {
	jwtSecret, err := loadJWTSecret()
	if err != nil {
		jwtSecret = config.JWTSecret
	}

	if jwtSecret == "" {
		fmt.Println("WARNING: JWT secret is not configured")
	}

	ignorePaths := make([]*regexp.Regexp, 0)
	for _, regex := range config.IgnorePathRegexes {
		compiledRegex, err := regexp.Compile(regex)
		if err == nil {
			ignorePaths = append(ignorePaths, compiledRegex)
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			for _, regex := range ignorePaths {
				if regex.MatchString(path) {
					return next(c)
				}
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format")
			}

			tokenString := tokenParts[1]

			token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
			}

			if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
				c.Set("user", claims)
				return next(c)
			}

			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
		}
	}
}

func GenerateToken(userID, username, role string, secret string, expiry time.Duration) (string, error) {
	if secret == "" {
		var err error
		secret, err = loadJWTSecret()
		if err != nil {
			return "", fmt.Errorf("no JWT secret available: %w", err)
		}
	}

	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
