#!/bin/bash

set -e

echo "🚀 GCP Cost Monitor - Setup Script"
echo "===================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisites() {
    echo "📋 Checking prerequisites..."
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Go installed${NC}"
    
    if ! command -v gcloud &> /dev/null; then
        echo -e "${YELLOW}⚠️  gcloud CLI not found (optional but recommended)${NC}"
    else
        echo -e "${GREEN}✅ gcloud CLI installed${NC}"
    fi
}

# Setup environment
setup_env() {
    echo ""
    echo "🔧 Setting up environment..."
    
    if [ ! -f .env ]; then
        cp .env.example .env
        echo -e "${GREEN}✅ Created .env from .env.example${NC}"
        echo "   Edit .env with your configuration"
    else
        echo -e "${YELLOW}⚠️  .env already exists${NC}"
    fi
}

# Build binary
build_binary() {
    echo ""
    echo "🔨 Building binary..."
    
    go mod download
    go build -o gcp-cost-monitor main.go gcp.go analyzer.go discord.go models.go
    
    chmod +x gcp-cost-monitor
    echo -e "${GREEN}✅ Binary built: ./gcp-cost-monitor${NC}"
}

# Setup cron
setup_cron() {
    echo ""
    echo "⏰ Setting up cron job..."
    echo ""
    echo "To run daily at 8 AM, add this to crontab:"
    echo ""
    
    FULL_PATH=$(pwd)
    echo -e "${YELLOW}0 8 * * * cd $FULL_PATH && ./gcp-cost-monitor >> /var/log/gcp-cost-monitor.log 2>&1${NC}"
    echo ""
    echo "Run 'crontab -e' and paste the line above"
}

# Test configuration
test_config() {
    echo ""
    echo "🧪 Testing configuration..."
    echo ""
    
    if [ -f .env ]; then
        source .env
        
        if [ -z "$GCP_PROJECT_ID" ]; then
            echo -e "${RED}❌ GCP_PROJECT_ID not set in .env${NC}"
            return 1
        fi
        
        if [ -z "$BILLING_TABLE" ]; then
            echo -e "${RED}❌ BILLING_TABLE not set in .env${NC}"
            return 1
        fi
        
        echo -e "${GREEN}✅ Configuration looks good${NC}"
        echo "   Project: $GCP_PROJECT_ID"
        echo "   Billing Table: $BILLING_TABLE"
        
        echo ""
        echo "💡 Run a dry-run test:"
        echo -e "${YELLOW}./gcp-cost-monitor -dry-run${NC}"
    else
        echo -e "${RED}❌ .env not found${NC}"
        return 1
    fi
}

# Main
main() {
    check_prerequisites
    setup_env
    build_binary
    setup_cron
    
    echo ""
    echo "===================================="
    echo -e "${GREEN}✅ Setup completed!${NC}"
    echo "===================================="
    echo ""
    echo "📖 Next steps:"
    echo "1. Edit .env with your GCP and Discord credentials"
    echo "2. Run: ./gcp-cost-monitor -dry-run"
    echo "3. Setup cron job (see above)"
    echo ""
}

main "$@"
