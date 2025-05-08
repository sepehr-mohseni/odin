#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configure URLs
GATEWAY_URL="http://localhost:8080"
USERS_URL="http://localhost:8081"
PRODUCTS_URL="http://localhost:8083"
ORDERS_URL="http://localhost:8084"
CATEGORIES_URL="http://localhost:8085"

# Sample JWT token - Replace this with an actual token for testing
# You can generate one using the jwt-generator.go tool
TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNyLTAwMSIsInVzZXJuYW1lIjoiam9obl9kb2UiLCJyb2xlIjoidXNlciIsImV4cCI6MTcxOTQ2MzkyOSwiaWF0IjoxNzE5NDYwMzI5fQ.WD6nbfa0K9VkFLEF9pWYZXQweQSU0IF6oKkdNULMpqE"
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}    Odin API Gateway Test Script    ${NC}"
echo -e "${GREEN}=========================================${NC}"

# Function to make HTTP requests and print responses
call_api() {
  local method=$1
  local url=$2
  local auth=$3
  local data=$4

  echo -e "\n${BLUE}[TEST]${NC} $method $url"

  local auth_header=""
  if [ ! -z "$auth" ]; then
    auth_header="-H \"Authorization: $auth\""
  fi

  local data_arg=""
  if [ ! -z "$data" ]; then
    data_arg="-d '$data' -H \"Content-Type: application/json\""
  fi

  local cmd="curl -s -X $method $auth_header $data_arg $url"
  local response=$(eval $cmd)

  echo -e "${YELLOW}Response:${NC}\n$response\n"
}

# Check if services are running
echo -e "\n${BLUE}Checking if services are running...${NC}"

if ! curl -s -o /dev/null -w "%{http_code}" $GATEWAY_URL/health | grep -q "200"; then
  echo -e "${RED}API Gateway is not running. Please start it before running tests.${NC}"
  exit 1
fi

# Test API gateway endpoints
echo -e "\n${GREEN}Testing direct service endpoints:${NC}"

# Test Users Service
echo -e "\n${YELLOW}Testing Users Service:${NC}"
call_api "GET" "$USERS_URL/api/users"
call_api "GET" "$USERS_URL/api/users/usr-001" "$TOKEN"
call_api "POST" "$USERS_URL/api/users" "$TOKEN" "{\"username\":\"test_user\",\"email\":\"test@example.com\",\"name\":\"Test User\"}"

# Test Products Service
echo -e "\n${YELLOW}Testing Products Service:${NC}"
call_api "GET" "$PRODUCTS_URL/api/products"
call_api "GET" "$PRODUCTS_URL/api/products/prod-001"
call_api "POST" "$PRODUCTS_URL/api/products" "$TOKEN" "{\"name\":\"Test Product\",\"price\":99.99,\"categoryId\":\"cat-001\",\"inStock\":true}"

# Test Orders Service
echo -e "\n${YELLOW}Testing Orders Service:${NC}"
call_api "GET" "$ORDERS_URL/api/orders" "$TOKEN"
call_api "GET" "$ORDERS_URL/api/orders/ord-001" "$TOKEN"
call_api "POST" "$ORDERS_URL/api/orders" "$TOKEN" "{\"userId\":\"usr-001\",\"items\":[{\"productId\":\"prod-001\",\"quantity\":1,\"price\":999.99}]}"

# Test Categories Service
echo -e "\n${YELLOW}Testing Categories Service:${NC}"
call_api "GET" "$CATEGORIES_URL/api/categories"
call_api "GET" "$CATEGORIES_URL/api/categories/cat-001"

# Test Gateway Routes
echo -e "\n${GREEN}Testing API Gateway routes:${NC}"

# Users via Gateway
echo -e "\n${YELLOW}Testing Users via Gateway:${NC}"
call_api "GET" "$GATEWAY_URL/api/users"
call_api "GET" "$GATEWAY_URL/api/users/usr-001" "$TOKEN"

# Products via Gateway
echo -e "\n${YELLOW}Testing Products via Gateway:${NC}"
call_api "GET" "$GATEWAY_URL/api/products"
call_api "GET" "$GATEWAY_URL/api/products/prod-001"

# Orders via Gateway
echo -e "\n${YELLOW}Testing Orders via Gateway:${NC}"
call_api "GET" "$GATEWAY_URL/api/orders" "$TOKEN"

# Categories via Gateway
echo -e "\n${YELLOW}Testing Categories via Gateway:${NC}"
call_api "GET" "$GATEWAY_URL/api/categories"

# Test Gateway Health Endpoints
echo -e "\n${YELLOW}Testing Gateway Health Endpoints:${NC}"
call_api "GET" "$GATEWAY_URL/health"
call_api "GET" "$GATEWAY_URL/metrics"

echo -e "\n${GREEN}Tests completed!${NC}"
