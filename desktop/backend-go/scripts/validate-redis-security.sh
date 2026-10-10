#!/bin/bash
set -e

# Redis Security Validation Script
# This script validates Redis security configuration for BusinessOS Go backend

echo "========================================="
echo "Redis Security Validation"
echo "========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}✗ .env file not found${NC}"
    echo "  Create .env from .env.example"
    exit 1
fi

# Load environment variables
export $(grep -v '^#' .env | xargs)

# Function to check if a variable is set
check_var() {
    local var_name=$1
    local var_value=${!var_name}
    local required=$2
    local min_length=$3

    if [ -z "$var_value" ]; then
        if [ "$required" = "true" ]; then
            echo -e "${RED}✗ $var_name is not set${NC}"
            return 1
        else
            echo -e "${YELLOW}⚠ $var_name is not set (optional)${NC}"
            return 0
        fi
    fi

    if [ ! -z "$min_length" ]; then
        if [ ${#var_value} -lt $min_length ]; then
            echo -e "${YELLOW}⚠ $var_name is set but shorter than recommended ($min_length characters)${NC}"
            return 1
        fi
    fi

    echo -e "${GREEN}✓ $var_name is configured${NC}"
    return 0
}

# Function to check weak passwords
check_weak_password() {
    local var_name=$1
    local var_value=${!var_name}
    local weak_patterns=("password" "123456" "changeme" "secret" "admin")

    for pattern in "${weak_patterns[@]}"; do
        if [[ "$var_value" == *"$pattern"* ]]; then
            if [ "$ENVIRONMENT" = "production" ]; then
                echo -e "${RED}✗ $var_name contains weak pattern: $pattern (CRITICAL in production)${NC}"
                return 1
            else
                echo -e "${YELLOW}⚠ $var_name contains weak pattern: $pattern (OK for dev)${NC}"
                return 0
            fi
        fi
    done

    return 0
}

# Validation counters
errors=0
warnings=0

echo ""
echo "1. Checking Redis URL..."
if check_var "REDIS_URL" "true"; then
    # Check if URL scheme is appropriate for TLS setting
    if [ "$REDIS_TLS_ENABLED" = "true" ]; then
        if [[ ! "$REDIS_URL" =~ ^rediss:// ]]; then
            echo -e "${RED}✗ REDIS_TLS_ENABLED=true but URL doesn't use rediss:// scheme${NC}"
            ((errors++))
        else
            echo -e "${GREEN}✓ Redis URL uses secure rediss:// scheme${NC}"
        fi
    else
        if [[ "$REDIS_URL" =~ ^rediss:// ]]; then
            echo -e "${YELLOW}⚠ Redis URL uses rediss:// but REDIS_TLS_ENABLED=false${NC}"
            ((warnings++))
        fi
    fi
else
    ((errors++))
fi

echo ""
echo "2. Checking Redis Password..."
if check_var "REDIS_PASSWORD" "true" 16; then
    check_weak_password "REDIS_PASSWORD" || ((warnings++))
else
    if [ "$ENVIRONMENT" = "production" ]; then
        ((errors++))
    else
        ((warnings++))
    fi
fi

echo ""
echo "3. Checking Redis TLS Configuration..."
if [ "$ENVIRONMENT" = "production" ]; then
    if [ "$REDIS_TLS_ENABLED" != "true" ]; then
        echo -e "${RED}✗ REDIS_TLS_ENABLED should be true in production${NC}"
        ((errors++))
    else
        echo -e "${GREEN}✓ Redis TLS is enabled for production${NC}"
    fi
else
    if [ "$REDIS_TLS_ENABLED" = "true" ]; then
        echo -e "${YELLOW}⚠ Redis TLS enabled in development (ensure certs are configured)${NC}"
        ((warnings++))
    else
        echo -e "${GREEN}✓ Redis TLS disabled for local development${NC}"
    fi
fi

echo ""
echo "4. Checking HMAC Secret..."
if check_var "REDIS_KEY_HMAC_SECRET" "true" 32; then
    check_weak_password "REDIS_KEY_HMAC_SECRET" || ((warnings++))

    # Check if it's a development placeholder
    if [[ "$REDIS_KEY_HMAC_SECRET" == *"dev"* ]] || [[ "$REDIS_KEY_HMAC_SECRET" == *"change"* ]]; then
        if [ "$ENVIRONMENT" = "production" ]; then
            echo -e "${RED}✗ REDIS_KEY_HMAC_SECRET appears to be a placeholder in production${NC}"
            ((errors++))
        else
            echo -e "${YELLOW}⚠ REDIS_KEY_HMAC_SECRET is a placeholder (generate strong value for production)${NC}"
            ((warnings++))
        fi
    fi
else
    if [ "$ENVIRONMENT" = "production" ]; then
        ((errors++))
    else
        ((warnings++))
    fi
fi

echo ""
echo "5. Checking Environment..."
echo "   Environment: $ENVIRONMENT"
if [ "$ENVIRONMENT" = "production" ]; then
    echo -e "${GREEN}✓ Running in production mode${NC}"
else
    echo -e "${GREEN}✓ Running in development mode${NC}"
fi

echo ""
echo "6. Testing Redis Connection..."
if command -v docker &> /dev/null; then
    if docker ps | grep -q businessos-redis; then
        echo -e "${GREEN}✓ Redis container is running${NC}"

        # Try to ping Redis
        if docker exec businessos-redis redis-cli -a "$REDIS_PASSWORD" ping &> /dev/null; then
            echo -e "${GREEN}✓ Redis authentication successful${NC}"
        else
            echo -e "${RED}✗ Redis authentication failed (check password)${NC}"
            ((errors++))
        fi
    else
        echo -e "${YELLOW}⚠ Redis container not running (start with: docker-compose up -d redis)${NC}"
        ((warnings++))
    fi
else
    echo -e "${YELLOW}⚠ Docker not available (skipping connection test)${NC}"
    ((warnings++))
fi

# Summary
echo ""
echo "========================================="
echo "Validation Summary"
echo "========================================="
echo "Errors: $errors"
echo "Warnings: $warnings"
echo ""

if [ $errors -gt 0 ]; then
    echo -e "${RED}✗ Validation failed with $errors error(s)${NC}"
    echo "Please fix the errors before deploying to production"
    exit 1
elif [ $warnings -gt 0 ]; then
    echo -e "${YELLOW}⚠ Validation passed with $warnings warning(s)${NC}"
    echo "Review warnings and fix before deploying to production"
    exit 0
else
    echo -e "${GREEN}✓ All validations passed!${NC}"
    echo "Redis security configuration is properly set up"
    exit 0
fi
