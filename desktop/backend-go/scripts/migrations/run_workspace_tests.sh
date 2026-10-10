#!/bin/bash

# Workspace Integration Tests Runner
# This script runs all workspace-related integration tests

set -e

echo "======================================================================"
echo "Workspace Feature Integration Tests"
echo "======================================================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if database is available
echo -e "${YELLOW}Checking database connection...${NC}"
if ! psql "${DATABASE_URL:-postgresql://localhost:5432/businessos}" -c '\q' 2>/dev/null; then
    echo -e "${RED}ERROR: Database is not available${NC}"
    echo "Please ensure PostgreSQL is running and DATABASE_URL is set"
    echo "Example: export DATABASE_URL=postgresql://user:password@localhost:5432/businessos"
    exit 1
fi
echo -e "${GREEN}Database connection successful${NC}"
echo ""

# Run migrations
echo -e "${YELLOW}Running database migrations...${NC}"
go run cmd/migrations/main.go up
echo -e "${GREEN}Migrations completed${NC}"
echo ""

# Test categories
echo "======================================================================"
echo "Running Test Suites"
echo "======================================================================"
echo ""

# 1. Workspace Service Tests
echo -e "${YELLOW}1. Workspace Service Tests${NC}"
echo "   Testing workspace creation, CRUD, roles, and members"
go test -v ./internal/services -run TestWorkspace -timeout 5m
echo ""

# 2. Memory Hierarchy Tests
echo -e "${YELLOW}2. Memory Hierarchy Service Tests${NC}"
echo "   Testing workspace vs private memory isolation"
go test -v ./internal/services -run TestMemoryHierarchy -timeout 5m
echo ""

# 3. Workspace Handler Tests
echo -e "${YELLOW}3. Workspace Handler Tests${NC}"
echo "   Testing HTTP API endpoints"
go test -v ./internal/handlers -run TestWorkspace -timeout 5m
go test -v ./internal/handlers -run TestAddMember -timeout 5m
go test -v ./internal/handlers -run TestUpdateMemberRole -timeout 5m
go test -v ./internal/handlers -run TestRemoveMember -timeout 5m
echo ""

# 4. Role Context Tests
echo -e "${YELLOW}4. Role Context Tests${NC}"
echo "   Testing permission checking and hierarchy"
go test -v ./internal/services -run TestRoleContext -timeout 5m
echo ""

# 5. Access Control Tests
echo -e "${YELLOW}5. Memory Access Control Tests${NC}"
echo "   Testing memory visibility and sharing"
go test -v ./internal/services -run TestMemoryAccessControl -timeout 5m
echo ""

# Summary
echo "======================================================================"
echo -e "${GREEN}All Workspace Integration Tests Completed${NC}"
echo "======================================================================"
echo ""
echo "Test Coverage:"
echo "  ✓ Workspace creation with 6 default roles"
echo "  ✓ Owner automatically added as first member"
echo "  ✓ Member management (add, update role, remove)"
echo "  ✓ Role hierarchy enforcement"
echo "  ✓ Permission checking"
echo "  ✓ Workspace vs private memory isolation"
echo "  ✓ Memory sharing with specific users"
echo "  ✓ Access control enforcement"
echo "  ✓ HTTP API endpoints"
echo "  ✓ Role context for agents"
echo ""
echo "Next Steps:"
echo "  - Review test output for any failures"
echo "  - Check test coverage: go test -cover ./internal/services ./internal/handlers"
echo "  - Run specific test: go test -v -run TestSpecificName ./internal/services"
echo ""
