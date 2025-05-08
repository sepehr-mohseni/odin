package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/yaml.v3"
)

type TokenClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type AuthSecrets struct {
	JWTSecret string `yaml:"jwtSecret"`
}

func loadJWTSecret() (string, error) {
	currentDir, _ := os.Getwd()
	scriptDir := filepath.Join(currentDir, "auth")

	secretsPaths := []string{
		"auth_secrets.yaml",
		filepath.Join(scriptDir, "auth_secrets.yaml"),
		filepath.Join(currentDir, "auth_secrets.yaml"),
		filepath.Join(currentDir, "../config/auth_secrets.yaml"),
		filepath.Join(currentDir, "../../config/auth_secrets.yaml"),
	}

	fmt.Println("Current directory:", currentDir)

	for _, path := range secretsPaths {
		fmt.Printf("Checking for auth_secrets.yaml at: %s\n", path)

		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("Found but couldn't read %s: %v\n", path, err)
				continue
			}

			var secrets AuthSecrets
			if err := yaml.Unmarshal(data, &secrets); err != nil {
				fmt.Printf("Found but couldn't parse %s: %v\n", path, err)
				continue
			}

			if secrets.JWTSecret != "" {
				return secrets.JWTSecret, nil
			} else {
				fmt.Printf("Found %s but it doesn't contain a JWT secret\n", path)
			}
		}
	}

	if envSecret := os.Getenv("ODIN_JWT_SECRET"); envSecret != "" {
		fmt.Println("Using JWT secret from environment variable ODIN_JWT_SECRET")
		return envSecret, nil
	}

	fmt.Println("\nERROR: Couldn't load JWT secret from any source")
	fmt.Println("Please create an auth_secrets.yaml file in the current directory with:")
	fmt.Println("jwtSecret: your-secret-here")
	fmt.Println("\nOR run with:")
	fmt.Println("go run auth/jwt-generator.go -secret=your-secret-here")

	return "", fmt.Errorf("couldn't load JWT secret from any source")
}

func main() {
	secretFromFile, _ := loadJWTSecret()

	secret := flag.String("secret", secretFromFile, "JWT secret key (will use from config file if found)")
	userID := flag.String("userid", "usr-001", "User ID")
	username := flag.String("username", "john_doe", "Username")
	role := flag.String("role", "user", "User role")
	expiry := flag.Duration("expiry", time.Hour, "Token expiry duration")
	flag.Parse()

	if *secret == "" {
		log.Fatal("JWT secret is required. Provide it with -secret flag or in auth_secrets.yaml")
	}

	claims := &TokenClaims{
		UserID:   *userID,
		Username: *username,
		Role:     *role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(*expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(*secret))
	if err != nil {
		log.Fatalf("Error generating token: %v", err)
	}

	fmt.Printf("Token: %s\n", tokenString)
	fmt.Printf("Authorization header: Bearer %s\n", tokenString)
	fmt.Printf("Expiry: %v\n", time.Now().Add(*expiry).Format(time.RFC3339))
}
