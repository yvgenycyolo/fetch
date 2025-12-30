#!/bin/bash

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}URL Fetch Service - Integration Test${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Check if server is running
echo -e "${BLUE}1. Checking if server is running...${NC}"
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓ Server is running${NC}\n"
else
    echo -e "${RED}✗ Server is not running. Please start it with: go run main.go${NC}"
    exit 1
fi

# Clear old results for clean test (optional, comment out if you want to keep old results)
echo -e "${BLUE}2. Clearing old results for clean test...${NC}"
CLEAR_RESPONSE=$(curl -s -X POST http://localhost:8080/admin/clear)
CLEARED_COUNT=$(echo "$CLEAR_RESPONSE" | jq -r '.results_cleared // 0')
echo -e "${GREEN}✓ Cleared ${CLEARED_COUNT} old results${NC}\n"

# Test POST request
echo -e "${BLUE}3. Submitting URLs for fetching...${NC}"
RESPONSE=$(curl -s -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://example.com",
      "https://httpbin.org/html",
      "http://cnn.com",
      "https://google.com",
      "http://invalid-domain-that-definitely-does-not-exist-12345.com"
    ]
  }')

echo "$RESPONSE" | jq '.'
echo -e "${GREEN}✓ URLs submitted${NC}\n"

# Wait for fetching to complete
echo -e "${BLUE}4. Waiting for URLs to be fetched (3 seconds)...${NC}"
sleep 3
echo -e "${GREEN}✓ Wait complete${NC}\n"

# Test GET request
echo -e "${BLUE}5. Retrieving fetch results...${NC}"
RESULTS=$(curl -s http://localhost:8080/fetch)

# Show full response for debugging
echo -e "${BLUE}Full Response:${NC}"
echo "$RESULTS" | jq '.'
echo ""

# Show summary
echo -e "${BLUE}6. Summary:${NC}"
TOTAL=$(echo "$RESULTS" | jq '.total_urls')
SUCCESS=$(echo "$RESULTS" | jq '.success_count')
FAILED=$(echo "$RESULTS" | jq '.failed_count')
PENDING=$(echo "$RESULTS" | jq '.pending_count')

echo -e "   Total URLs:     ${BLUE}${TOTAL}${NC}"
echo -e "   Successful:     ${GREEN}${SUCCESS}${NC}"
echo -e "   Failed:         ${RED}${FAILED}${NC}"
echo -e "   Pending:        ${BLUE}${PENDING}${NC}"
echo ""

# Show individual results
echo -e "${BLUE}7. Individual Results (formatted):${NC}"
echo "$RESULTS" | jq -r '.results[] | 
  "\(.status | if . == "success" then "✓" elif . == "failed" then "✗" else "⋯" end) \(.url)
    Status: \(.status)
    Status Code: \(.status_code // "N/A")
    Size: \(.content_length // 0) bytes
    Redirects: \(.redirect_count // 0)
    Duration: \(.duration // "N/A")
    Final URL: \(.final_url // .url)
    Error: \(.error // "None")
    Created: \(.created_at)
    Fetched: \(.fetched_at // "Not yet")"'
echo ""

# Show raw JSON for debugging
echo -e "${BLUE}7b. Raw Result Objects (for debugging):${NC}"
echo "$RESULTS" | jq '.results[]' | head -50
echo ""

# Show redirect details
echo -e "${BLUE}8. Redirect Details:${NC}"
echo "$RESULTS" | jq -r '.results[] | select(.redirect_count != null and .redirect_count > 0) | "  \(.url) → \(.final_url // "N/A") (\(.redirect_count) redirect\(if .redirect_count > 1 then "s" else "" end))"'
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Test completed successfully!${NC}"
echo -e "${BLUE}========================================${NC}"

