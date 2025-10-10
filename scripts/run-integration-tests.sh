#!/bin/bash

# SAGE ADK Integration Tests Runner
# This script starts test containers and runs integration tests

set -e

echo "🚀 SAGE ADK Integration Tests"
echo "=============================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Start test containers
echo -e "${YELLOW}📦 Starting test containers...${NC}"
docker-compose -f docker-compose.test.yml up -d

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
sleep 5

# Check Redis
echo -n "Checking Redis... "
if docker exec sage-test-redis redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo -e "${RED}❌ Redis is not ready${NC}"
    docker-compose -f docker-compose.test.yml logs redis
    docker-compose -f docker-compose.test.yml down
    exit 1
fi

# Check PostgreSQL
echo -n "Checking PostgreSQL... "
if docker exec sage-test-postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo -e "${RED}❌ PostgreSQL is not ready${NC}"
    docker-compose -f docker-compose.test.yml logs postgres
    docker-compose -f docker-compose.test.yml down
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Test environment ready${NC}"
echo ""

# Run integration tests
echo -e "${YELLOW}🧪 Running integration tests...${NC}"
echo ""

# Set test environment variables
export REDIS_URL="localhost:6381"
export POSTGRES_URL="postgres://postgres:test@localhost:5434/postgres?sslmode=disable"

# Run tests with integration tag
if go test -tags=integration -v ./storage/ -run="Integration|Concurrent|LargeData"; then
    echo ""
    echo -e "${GREEN}✓ All integration tests passed!${NC}"
    TEST_RESULT=0
else
    echo ""
    echo -e "${RED}❌ Some integration tests failed${NC}"
    TEST_RESULT=1
fi

# Cleanup option
echo ""
read -p "Stop test containers? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}🧹 Stopping test containers...${NC}"
    docker-compose -f docker-compose.test.yml down
    echo -e "${GREEN}✓ Cleanup complete${NC}"
else
    echo -e "${YELLOW}Test containers are still running.${NC}"
    echo "To stop them later, run:"
    echo "  docker-compose -f docker-compose.test.yml down"
fi

echo ""
if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}✨ Integration tests completed successfully!${NC}"
else
    echo -e "${RED}💥 Integration tests failed. Check the output above.${NC}"
fi

exit $TEST_RESULT
