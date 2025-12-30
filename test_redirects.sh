#!/bin/bash

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}URL Fetch Service - Redirect Test Suite${NC}"
echo -e "${CYAN}========================================${NC}\n"

# Check if server is running
echo -e "${BLUE}1. Checking if server is running...${NC}"
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓ Server is running${NC}\n"
else
    echo -e "${RED}✗ Server is not running. Please start it with: go run main.go${NC}"
    exit 1
fi

# Clear old results for clean test
echo -e "${BLUE}2. Clearing old results for clean test...${NC}"
CLEAR_RESPONSE=$(curl -s -X POST http://localhost:8080/admin/clear)
CLEARED_COUNT=$(echo "$CLEAR_RESPONSE" | jq -r '.results_cleared // 0')
echo -e "${GREEN}✓ Cleared ${CLEARED_COUNT} old results${NC}\n"

# Test various redirect scenarios
echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 1: HTTP to HTTPS Redirects${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Submitting HTTP URLs that redirect to HTTPS...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "http://cnn.com",
      "http://github.com",
      "http://stackoverflow.com"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 5 seconds for fetches to complete...${NC}"
sleep 5

echo -e "\n${BLUE}Results (last 3):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show detailed results
echo "$RESULTS" | jq '.results[-3:]'

echo -e "\n${BLUE}Results Summary (last 3):${NC}"
echo "$RESULTS" | jq '.results[-3:] | .[] | {
  original: .url,
  final: .final_url,
  redirects: .redirect_count,
  status: .status,
  status_code: .status_code,
  size: .content_length,
  duration: .duration,
  error: .error
}'

echo -e "\n${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 2: Domain Redirects (www)${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Submitting URLs that redirect to www subdomain...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://google.com",
      "https://youtube.com",
      "https://reddit.com"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 5 seconds for fetches to complete...${NC}"
sleep 5

echo -e "\n${BLUE}Results (last 3):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show detailed results
echo "$RESULTS" | jq '.results[-3:]'

echo -e "\n${BLUE}Results Summary (last 3):${NC}"
echo "$RESULTS" | jq '.results[-3:] | .[] | {
  original: .url,
  final: .final_url,
  redirects: .redirect_count,
  status: .status,
  duration: .duration,
  error: .error
}'

echo -e "\n${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 3: Short URL Redirects${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Testing URL shortener services...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/redirect/1",
      "https://httpbin.org/redirect/2",
      "https://httpbin.org/redirect/3"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 5 seconds for fetches to complete...${NC}"
sleep 5

echo -e "\n${BLUE}Results (last 3):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show detailed results  
echo "$RESULTS" | jq '.results[-3:]'

echo -e "\n${BLUE}Results Summary (last 3):${NC}"
echo "$RESULTS" | jq '.results[-3:] | .[] | {
  original: .url,
  final: .final_url,
  redirects: .redirect_count,
  status: .status,
  status_code: .status_code,
  error: .error
}'

echo -e "\n${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 4: Different Redirect Status Codes${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Testing 301, 302, 307, 308 redirects...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/status/301",
      "https://httpbin.org/status/302",
      "https://httpbin.org/redirect-to?url=https://httpbin.org/html&status_code=307",
      "https://httpbin.org/redirect-to?url=https://httpbin.org/html&status_code=308"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 5 seconds for fetches to complete...${NC}"
sleep 5

echo -e "\n${BLUE}Results (last 4):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show full objects
echo "$RESULTS" | jq '.results[-4:]'

echo -e "\n${BLUE}Results Summary (last 4):${NC}"
echo "$RESULTS" | jq '.results[-4:] | .[] | {
  original: .url,
  final: .final_url,
  redirects: .redirect_count,
  status: .status,
  error: .error
}'

echo -e "\n${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 5: Redirect Chains${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Testing multi-hop redirect chains...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/redirect/5",
      "https://httpbin.org/redirect/8"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 6 seconds for fetches to complete...${NC}"
sleep 6

echo -e "\n${BLUE}Results (last 2):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show full objects
echo "$RESULTS" | jq '.results[-2:]'

echo -e "\n${BLUE}Results Summary (last 2):${NC}"
echo "$RESULTS" | jq '.results[-2:] | .[] | {
  original: .url,
  final: .final_url,
  redirects: .redirect_count,
  status: .status,
  duration: .duration,
  error: .error
}'

echo -e "\n${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 6: Too Many Redirects (Should Fail)${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Testing redirect limit (max 10)...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/redirect/15",
      "https://httpbin.org/redirect/20"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 6 seconds for fetches to complete...${NC}"
sleep 6

echo -e "\n${BLUE}Results (should show failures, last 2):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show full objects for debugging
echo "$RESULTS" | jq '.results[-2:]'

echo -e "\n${BLUE}Results Summary (last 2):${NC}"
echo "$RESULTS" | jq '.results[-2:] | .[] | {
  original: .url,
  redirects: .redirect_count,
  status: .status,
  error: .error
}'

echo -e "\n${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 7: Absolute vs Relative Redirects${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Testing absolute and relative redirect paths...${NC}"
curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/absolute-redirect/2",
      "https://httpbin.org/relative-redirect/2"
    ]
  }' | jq '.'

echo -e "\n${YELLOW}Waiting 5 seconds for fetches to complete...${NC}"
sleep 5

echo -e "\n${BLUE}Results (last 2):${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show full objects
echo "$RESULTS" | jq '.results[-2:]'

echo -e "\n${BLUE}Results Summary (last 2):${NC}"
echo "$RESULTS" | jq '.results[-2:] | .[] | {
  original: .url,
  final: .final_url,
  redirects: .redirect_count,
  status: .status,
  error: .error
}'

# Summary
echo -e "\n${CYAN}========================================${NC}"
echo -e "${CYAN}Summary Statistics${NC}"
echo -e "${CYAN}========================================${NC}\n"

RESULTS=$(curl -s http://localhost:8080/fetch)

echo -e "${BLUE}Overall Statistics:${NC}"
echo "$RESULTS" | jq '{
  total_urls: .total_urls,
  successful: .success_count,
  failed: .failed_count,
  pending: .pending_count
}'

echo -e "\n${BLUE}Redirect Statistics:${NC}"
echo "$RESULTS" | jq '[.results[] | select(.redirect_count != null and .redirect_count > 0)] | 
  if length > 0 then {
    total_with_redirects: length,
    avg_redirects: (map(.redirect_count) | add / length),
    max_redirects: (map(.redirect_count) | max),
    min_redirects: (map(.redirect_count) | min)
  } else {
    total_with_redirects: 0,
    message: "No results with redirects found"
  } end'

echo -e "\n${BLUE}Top Redirected URLs:${NC}"
echo "$RESULTS" | jq -r '[.results[] | select(.redirect_count != null and .redirect_count > 0)] | 
  sort_by(.redirect_count) | reverse | .[:5] | .[] | 
  "  \(.redirect_count) redirects: \(.url) → \(.final_url // "N/A")"'

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Redirect test suite completed!${NC}"
echo -e "${GREEN}========================================${NC}\n"



