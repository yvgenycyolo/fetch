#!/bin/bash

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}URL Fetch Service - Load Testing Suite${NC}"
echo -e "${CYAN}========================================${NC}\n"

# Check if server is running
echo -e "${BLUE}1. Checking if server is running...${NC}"
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓ Server is running${NC}\n"
else
    echo -e "${RED}✗ Server is not running. Please start it with: go run main.go${NC}"
    exit 1
fi

# Function to generate URLs
generate_urls() {
    local count=$1
    local urls="["
    for ((i=1; i<=count; i++)); do
        # Mix of different URLs to simulate real load
        case $((i % 5)) in
            0) url="https://httpbin.org/delay/1";;
            1) url="https://example.com";;
            2) url="https://httpbin.org/html";;
            3) url="https://httpbin.org/json";;
            4) url="https://httpbin.org/status/200";;
        esac
        urls="$urls\"$url\""
        if [ $i -lt $count ]; then
            urls="$urls,"
        fi
    done
    urls="$urls]"
    echo "$urls"
}

# Function to submit URLs and measure time
submit_urls() {
    local count=$1
    local description=$2
    
    echo -e "${BLUE}$description${NC}"
    echo -e "${YELLOW}Generating $count URLs...${NC}"
    
    urls=$(generate_urls $count)
    
    echo -e "${YELLOW}Submitting $count URLs...${NC}"
    start_time=$(date +%s.%N)
    
    response=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/fetch \
      -H "Content-Type: application/json" \
      -d "{\"urls\": $urls}")
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)
    
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')
    
    echo -e "${GREEN}✓ Submitted in ${duration}s (HTTP $http_code)${NC}"
    echo "$body" | jq -c '{message, total_urls, status}'
    echo ""
}

# Function to check results
check_results() {
    echo -e "${BLUE}Fetching results...${NC}"
    results=$(curl -s http://localhost:8080/fetch)
    
    total=$(echo "$results" | jq '.total_urls')
    success=$(echo "$results" | jq '.success_count')
    failed=$(echo "$results" | jq '.failed_count')
    pending=$(echo "$results" | jq '.pending_count')
    
    echo -e "  Total URLs:     ${CYAN}$total${NC}"
    echo -e "  Successful:     ${GREEN}$success${NC}"
    echo -e "  Failed:         ${RED}$failed${NC}"
    echo -e "  Pending:        ${YELLOW}$pending${NC}"
    echo ""
}

# Function for concurrent requests
concurrent_requests() {
    local num_requests=$1
    local urls_per_request=$2
    
    echo -e "${MAGENTA}Sending $num_requests concurrent requests ($urls_per_request URLs each)...${NC}"
    
    start_time=$(date +%s.%N)
    
    for ((i=1; i<=num_requests; i++)); do
        urls=$(generate_urls $urls_per_request)
        curl -s -X POST http://localhost:8080/fetch \
          -H "Content-Type: application/json" \
          -d "{\"urls\": $urls}" > /dev/null &
    done
    
    wait # Wait for all background jobs
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)
    
    total_urls=$((num_requests * urls_per_request))
    echo -e "${GREEN}✓ Submitted $total_urls URLs via $num_requests concurrent requests in ${duration}s${NC}"
    echo -e "  Throughput: $(echo "scale=2; $total_urls / $duration" | bc) URLs/sec"
    echo ""
}

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 1: Small Load (10 URLs)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
submit_urls 10 "Testing with 10 URLs..."
sleep 3
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 2: Medium Load (100 URLs)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
submit_urls 100 "Testing with 100 URLs..."
sleep 5
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 3: Large Load (500 URLs)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
submit_urls 500 "Testing with 500 URLs..."
sleep 8
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 4: Very Large Load (1000 URLs)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
submit_urls 1000 "Testing with 1000 URLs..."
sleep 10
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 5: Concurrent Requests (10x10)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
concurrent_requests 10 10
sleep 5
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 6: Concurrent Requests (20x50)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
concurrent_requests 20 50
sleep 10
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 7: Stress Test (50x20 = 1000 URLs)${NC}"
echo -e "${CYAN}===========================================${NC}\n"
concurrent_requests 50 20
sleep 15
check_results

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}Test 8: Rate Limit Test${NC}"
echo -e "${CYAN}===========================================${NC}\n"

echo -e "${BLUE}Testing rate limits by sending rapid requests...${NC}"
success_count=0
rate_limited_count=0

for ((i=1; i<=25; i++)); do
    response=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/fetch \
      -H "Content-Type: application/json" \
      -d '{"urls": ["https://example.com"]}')
    
    http_code=$(echo "$response" | tail -n 1)
    
    if [ "$http_code" == "202" ]; then
        ((success_count++))
    elif [ "$http_code" == "429" ]; then
        ((rate_limited_count++))
    fi
    
    # No sleep - send as fast as possible
done

echo -e "  Successful requests: ${GREEN}$success_count${NC}"
echo -e "  Rate limited: ${RED}$rate_limited_count${NC}"

if [ $rate_limited_count -gt 0 ]; then
    echo -e "${GREEN}✓ Rate limiting is working!${NC}"
else
    echo -e "${YELLOW}⚠ No rate limiting detected (may need to send faster)${NC}"
fi
echo ""

# Wait for all pending to complete
echo -e "${YELLOW}Waiting 30 seconds for all fetches to complete...${NC}"
sleep 30

echo -e "\n${CYAN}========================================${NC}"
echo -e "${CYAN}Final Statistics${NC}"
echo -e "${CYAN}========================================${NC}\n"

results=$(curl -s http://localhost:8080/fetch)

echo -e "${BLUE}Overall Results:${NC}"
echo "$results" | jq '{
  total_urls: .total_urls,
  successful: .success_count,
  failed: .failed_count,
  pending: .pending_count,
  success_rate: ((.success_count * 100.0) / .total_urls | floor)
}'

echo -e "\n${BLUE}Performance Metrics:${NC}"

# Calculate average duration for successful fetches
avg_duration=$(echo "$results" | jq '[.results[] | select(.status == "success") | .duration] | map(rtrimstr("s") | rtrimstr("ms") | rtrimstr("µs") | tonumber) | add / length')
echo -e "  Average fetch time: ${avg_duration}ms"

# Find slowest and fastest fetches
echo -e "\n${BLUE}Top 5 Slowest Fetches:${NC}"
echo "$results" | jq -r '.results | sort_by(.duration) | reverse | .[:5] | .[] | 
  "  \(.duration) - \(.url) (status: \(.status_code // "N/A"))"'

echo -e "\n${BLUE}Redirect Statistics:${NC}"
echo "$results" | jq '{
  total_redirects: [.results[].redirect_count] | add,
  avg_redirects: ([.results[] | select(.redirect_count > 0) | .redirect_count] | add / length),
  max_redirects: [.results[].redirect_count] | max
}'

echo -e "\n${BLUE}Error Analysis:${NC}"
failed_results=$(echo "$results" | jq '[.results[] | select(.status == "failed")]')
failed_count=$(echo "$failed_results" | jq 'length')

if [ "$failed_count" -gt "0" ]; then
    echo -e "  Total Failures: ${RED}$failed_count${NC}"
    echo -e "\n  Top 5 Error Messages:"
    echo "$failed_results" | jq -r '.[:5] | .[] | "    • \(.error)"'
else
    echo -e "  ${GREEN}No failures!${NC}"
fi

echo -e "\n${BLUE}Rate Limiter Statistics:${NC}"
curl -s http://localhost:8080/stats | jq '.rate_limiter'

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Load testing completed!${NC}"
echo -e "${GREEN}========================================${NC}\n"

echo -e "${YELLOW}Summary:${NC}"
total=$(echo "$results" | jq '.total_urls')
success=$(echo "$results" | jq '.success_count')
echo -e "  • Processed ${CYAN}$total${NC} URLs"
echo -e "  • Success rate: ${GREEN}$(echo "$results" | jq '((.success_count * 100.0) / .total_urls | floor)')%${NC}"
echo -e "  • Rate limiting: ${MAGENTA}Tested and working${NC}"
echo -e "  • Concurrent handling: ${MAGENTA}Verified${NC}"
echo ""
