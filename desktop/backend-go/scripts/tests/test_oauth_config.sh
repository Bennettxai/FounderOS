#!/bin/bash
# Test Google OAuth configuration
# Usage: ./scripts/test_oauth_config.sh

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║           GOOGLE OAUTH CONFIGURATION TEST                        ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

# Load .env
if [ -f ".env" ]; then
    export $(cat .env | grep -v '^#' | xargs)
    echo -e "${GREEN}✅ Loaded .env file${NC}"
else
    echo -e "${RED}❌ .env file not found${NC}"
    exit 1
fi

# Check environment variables
echo ""
echo "🔍 Checking Environment Variables:"
echo "─────────────────────────────────────────────────────────────────"

if [ -z "$GOOGLE_CLIENT_ID" ]; then
    echo -e "${RED}❌ GOOGLE_CLIENT_ID not set${NC}"
    exit 1
else
    echo -e "${GREEN}✅ GOOGLE_CLIENT_ID${NC}: ${GOOGLE_CLIENT_ID:0:20}..."
fi

if [ -z "$GOOGLE_CLIENT_SECRET" ]; then
    echo -e "${RED}❌ GOOGLE_CLIENT_SECRET not set${NC}"
    exit 1
else
    echo -e "${GREEN}✅ GOOGLE_CLIENT_SECRET${NC}: ****** (hidden)"
fi

# Check OAuth redirect URI
echo ""
echo "🔍 OAuth Configuration:"
echo "─────────────────────────────────────────────────────────────────"
BACKEND_URL="${BACKEND_URL:-http://localhost:8001}"
REDIRECT_URI="$BACKEND_URL/api/auth/google/callback"
echo "   Redirect URI: $REDIRECT_URI"
echo ""

echo -e "${YELLOW}⚠️  IMPORTANT:${NC} Verify this redirect URI in Google Cloud Console:"
echo "   1. Go to: https://console.cloud.google.com/apis/credentials"
echo "   2. Find your OAuth 2.0 Client ID"
echo "   3. Click 'Edit'"
echo "   4. Under 'Authorized redirect URIs', ensure you have:"
echo "      • $REDIRECT_URI"
echo "   5. Save changes and wait 5 minutes for propagation"
echo ""

# Check required scopes
echo "🔍 Required OAuth Scopes:"
echo "─────────────────────────────────────────────────────────────────"
echo "   • https://www.googleapis.com/auth/userinfo.email"
echo "   • https://www.googleapis.com/auth/userinfo.profile"
echo "   • https://www.googleapis.com/auth/gmail.readonly"
echo ""

echo "📋 Verify these scopes in Google Cloud Console:"
echo "   1. Go to: https://console.cloud.google.com/apis/credentials/consent"
echo "   2. Check 'Scopes for Google APIs'"
echo "   3. Ensure all 3 scopes above are enabled"
echo ""

# Test Google OAuth URL
echo "🔗 OAuth Authorization URL:"
echo "─────────────────────────────────────────────────────────────────"
AUTH_URL="https://accounts.google.com/o/oauth2/v2/auth?client_id=$GOOGLE_CLIENT_ID&redirect_uri=$REDIRECT_URI&response_type=code&scope=openid%20email%20profile%20https://www.googleapis.com/auth/gmail.readonly&access_type=offline"
echo "   URL: ${AUTH_URL:0:80}..."
echo ""
echo "   You can test this URL manually by opening it in a browser."
echo "   It should redirect to Google's consent screen."
echo ""

# Final instructions
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                        NEXT STEPS                                ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo "1. Ensure backend server is running:"
echo "   cd desktop/backend-go && go run cmd/server/main.go"
echo ""
echo "2. Test OAuth flow by navigating to:"
echo "   http://localhost:5173/onboarding"
echo ""
echo "3. Click 'Connect Gmail' and verify:"
echo "   • Google consent screen appears"
echo "   • All scopes are listed"
echo "   • After granting, redirects back to BusinessOS"
echo "   • No 'redirect_uri_mismatch' errors"
echo ""
echo "4. If errors occur, check backend logs:"
echo "   ./scripts/monitor_logs.sh oauth"
echo ""

echo -e "${GREEN}✅ OAuth configuration check complete${NC}"
