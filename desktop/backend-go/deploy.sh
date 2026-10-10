#!/bin/bash
# Deploy BusinessOS backend to Google Cloud Run
# Usage: ./deploy.sh [setup|deploy|secrets]
#
# Commands:
#   setup   - First-time setup (enable APIs, create database, configure secrets)
#   secrets - Create/update secrets in Secret Manager
#   deploy  - Build and deploy to Cloud Run (default)

set -e

# Configuration - UPDATE THESE
PROJECT_ID="${GCP_PROJECT_ID:-your-project-id}"
REGION="us-central1"
SERVICE_NAME="businessos-api"
IMAGE_NAME="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"
DB_INSTANCE_NAME="businessos-db"
DB_NAME="business_os"

COMMAND="${1:-deploy}"

echo "=== BusinessOS Backend Deployment ==="
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo "Service: $SERVICE_NAME"
echo "Command: $COMMAND"
echo ""

# Check if gcloud is configured
if ! gcloud config get-value project &>/dev/null; then
    echo "Error: gcloud not configured. Run: gcloud init"
    exit 1
fi

# Set project
gcloud config set project $PROJECT_ID

setup_project() {
    echo "=== Setting up GCP Project ==="

    # Enable required APIs
    echo "Enabling required APIs..."
    gcloud services enable \
        cloudbuild.googleapis.com \
        run.googleapis.com \
        sqladmin.googleapis.com \
        secretmanager.googleapis.com \
        containerregistry.googleapis.com

    # Create Cloud SQL instance
    echo "Creating Cloud SQL instance (this may take a few minutes)..."
    if ! gcloud sql instances describe $DB_INSTANCE_NAME &>/dev/null; then
        gcloud sql instances create $DB_INSTANCE_NAME \
            --database-version=POSTGRES_15 \
            --tier=db-f1-micro \
            --region=$REGION \
            --storage-auto-increase \
            --deletion-protection

        # Set root password
        echo "Setting database password..."
        gcloud sql users set-password postgres \
            --instance=$DB_INSTANCE_NAME \
            --password="${DB_PASSWORD:-$(openssl rand -base64 24)}"
    else
        echo "Cloud SQL instance already exists"
    fi

    # Create database
    echo "Creating database..."
    gcloud sql databases create $DB_NAME --instance=$DB_INSTANCE_NAME 2>/dev/null || true

    echo ""
    echo "Setup complete! Now run: ./deploy.sh secrets"
}

create_secrets() {
    echo "=== Creating Secrets in Secret Manager ==="
    echo "You'll be prompted to enter values for each secret."
    echo ""

    # DATABASE_URL
    echo "Enter DATABASE_URL (format: postgres://user:pass@/dbname?host=/cloudsql/PROJECT:REGION:INSTANCE):"
    read -s DATABASE_URL
    echo "$DATABASE_URL" | gcloud secrets create DATABASE_URL --data-file=- 2>/dev/null || \
        echo "$DATABASE_URL" | gcloud secrets versions add DATABASE_URL --data-file=-
    echo "✓ DATABASE_URL saved"

    # GOOGLE_CLIENT_ID
    echo "Enter GOOGLE_CLIENT_ID:"
    read GOOGLE_CLIENT_ID
    echo "$GOOGLE_CLIENT_ID" | gcloud secrets create GOOGLE_CLIENT_ID --data-file=- 2>/dev/null || \
        echo "$GOOGLE_CLIENT_ID" | gcloud secrets versions add GOOGLE_CLIENT_ID --data-file=-
    echo "✓ GOOGLE_CLIENT_ID saved"

    # GOOGLE_CLIENT_SECRET
    echo "Enter GOOGLE_CLIENT_SECRET:"
    read -s GOOGLE_CLIENT_SECRET
    echo "$GOOGLE_CLIENT_SECRET" | gcloud secrets create GOOGLE_CLIENT_SECRET --data-file=- 2>/dev/null || \
        echo "$GOOGLE_CLIENT_SECRET" | gcloud secrets versions add GOOGLE_CLIENT_SECRET --data-file=-
    echo "✓ GOOGLE_CLIENT_SECRET saved"

    # SECRET_KEY
    echo "Generating SECRET_KEY..."
    SECRET_KEY=$(openssl rand -base64 32)
    echo "$SECRET_KEY" | gcloud secrets create SECRET_KEY --data-file=- 2>/dev/null || \
        echo "$SECRET_KEY" | gcloud secrets versions add SECRET_KEY --data-file=-
    echo "✓ SECRET_KEY saved"

    # ANTHROPIC_API_KEY (optional)
    echo "Enter ANTHROPIC_API_KEY (optional, press Enter to skip):"
    read -s ANTHROPIC_API_KEY
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        echo "$ANTHROPIC_API_KEY" | gcloud secrets create ANTHROPIC_API_KEY --data-file=- 2>/dev/null || \
            echo "$ANTHROPIC_API_KEY" | gcloud secrets versions add ANTHROPIC_API_KEY --data-file=-
        echo "✓ ANTHROPIC_API_KEY saved"
    fi

    # Grant Cloud Run access to secrets
    echo ""
    echo "Granting Cloud Run access to secrets..."
    PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
    SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

    for secret in DATABASE_URL GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET SECRET_KEY ANTHROPIC_API_KEY; do
        gcloud secrets add-iam-policy-binding $secret \
            --member="serviceAccount:$SERVICE_ACCOUNT" \
            --role="roles/secretmanager.secretAccessor" 2>/dev/null || true
    done

    echo ""
    echo "Secrets configured! Now run: ./deploy.sh deploy"
}

deploy_service() {
    # Build the Docker image
    echo "Building Docker image..."
    docker build -t $IMAGE_NAME:latest .

    # Push to Container Registry
    echo "Pushing to Container Registry..."
    docker push $IMAGE_NAME:latest

    # Get project number for service URL
    PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
    SERVICE_URL="https://${SERVICE_NAME}-${PROJECT_NUMBER}.${REGION}.run.app"

    # Deploy to Cloud Run
    echo "Deploying to Cloud Run..."
    gcloud run deploy $SERVICE_NAME \
        --image $IMAGE_NAME:latest \
        --region $REGION \
        --platform managed \
        --allow-unauthenticated \
        --add-cloudsql-instances "${PROJECT_ID}:${REGION}:${DB_INSTANCE_NAME}" \
        --memory 512Mi \
        --cpu 1 \
        --min-instances 0 \
        --max-instances 10 \
        --set-env-vars "ENVIRONMENT=production,SERVER_PORT=8080,AI_PROVIDER=anthropic,ENABLE_LOCAL_MODELS=false" \
        --set-secrets "DATABASE_URL=DATABASE_URL:latest,GOOGLE_CLIENT_ID=GOOGLE_CLIENT_ID:latest,GOOGLE_CLIENT_SECRET=GOOGLE_CLIENT_SECRET:latest,SECRET_KEY=SECRET_KEY:latest,ANTHROPIC_API_KEY=ANTHROPIC_API_KEY:latest" \
        --update-env-vars "GOOGLE_REDIRECT_URI=${SERVICE_URL}/api/auth/google/callback/login" \
        --update-env-vars "ALLOWED_ORIGINS=https://businessos.app,app://localhost,http://localhost:5173,http://localhost:5174"

    # Get the actual service URL
    SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format 'value(status.url)')

    echo ""
    echo "=== Deployment Complete ==="
    echo "Service URL: $SERVICE_URL"
    echo ""
    echo "Auth endpoints:"
    echo "  - Google OAuth: ${SERVICE_URL}/api/auth/google"
    echo "  - Email Sign Up: POST ${SERVICE_URL}/api/auth/sign-up/email"
    echo "  - Email Sign In: POST ${SERVICE_URL}/api/auth/sign-in/email"
    echo "  - Session: ${SERVICE_URL}/api/auth/session"
    echo ""
    echo "Don't forget to:"
    echo "1. Add ${SERVICE_URL}/api/auth/google/callback/login to Google OAuth redirect URIs"
    echo "2. Update your desktop app with this cloud URL: $SERVICE_URL"
}

case $COMMAND in
    setup)
        setup_project
        ;;
    secrets)
        create_secrets
        ;;
    deploy)
        deploy_service
        ;;
    *)
        echo "Unknown command: $COMMAND"
        echo "Usage: ./deploy.sh [setup|secrets|deploy]"
        exit 1
        ;;
esac
