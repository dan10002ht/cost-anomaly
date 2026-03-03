# GCP Cost Anomaly Detector

A Go-based tool that monitors GCP daily costs and sends detailed anomaly reports to Discord.

**Features:**
- ✅ Daily cost anomaly detection (20% threshold by default)
- ✅ Service-level cost breakdown
- ✅ Deep-dive BigQuery job analysis
- ✅ Intelligent recommendations
- ✅ Discord webhook integration
- ✅ Zero external dependencies (except GCP SDKs)

## Architecture

```
┌─────────────────────┐
│   main.go           │ ← Entry point + orchestration
└──────────┬──────────┘
           │
    ┌──────┴──────┬──────────┬──────────┐
    │             │          │          │
    ▼             ▼          ▼          ▼
 gcp.go      analyzer.go  discord.go  models.go
 (Queries)  (Detection)   (Webhook)   (Structs)
    │             │          │
    └──────┬──────┴────┬─────┘
           │           │
    ┌──────▼────┐   ┌──▼────────┐
    │ BigQuery  │   │  Discord   │
    │  Queries  │   │   Webhook  │
    └───────────┘   └────────────┘
```

## Setup

### 1. Prerequisites

- Go 1.21+
- GCP Project with:
  - BigQuery enabled
  - Billing export to BigQuery configured
  - Service account with appropriate permissions
- Discord server with webhook access

### 2. GCP Setup

#### Enable Billing Export to BigQuery

1. Go to **Cloud Console** → **Billing** → **Billing Export**
2. Click **Export to BigQuery**
3. Select dataset (default: `billing_data`)
4. Note the dataset and table names

#### Create Service Account

```bash
# Create service account
gcloud iam service-accounts create gcp-cost-monitor \
  --display-name "GCP Cost Monitor"

# Grant permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member=serviceAccount:gcp-cost-monitor@YOUR_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/bigquery.dataViewer

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member=serviceAccount:gcp-cost-monitor@YOUR_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/bigquery.jobUser

# Create and download key
gcloud iam service-accounts keys create key.json \
  --iam-account=gcp-cost-monitor@YOUR_PROJECT_ID.iam.gserviceaccount.com
```

### 3. Discord Setup

1. Go to your Discord server
2. **Server Settings** → **Webhooks** → **New Webhook**
3. Select target channel: `#1478341525703229532` (or your preferred channel)
4. Copy webhook URL

### 4. Clone and Build

```bash
# Clone
git clone https://github.com/yourusername/gcp-cost-monitor.git
cd gcp-cost-monitor

# Download dependencies
go mod download

# Build
go build -o gcp-cost-monitor main.go gcp.go analyzer.go discord.go models.go
```

### 5. Configure Environment

```bash
# Copy example env
cp .env.example .env

# Edit .env with your values
nano .env

# Load env vars
export $(cat .env | xargs)
```

Or set environment variables directly:

```bash
export GCP_PROJECT_ID="your-project-id"
export GOOGLE_APPLICATION_CREDENTIALS="./key.json"
export BILLING_TABLE="gcp_billing_export_v1_XXXXXX"
export DISCORD_WEBHOOK="https://discord.com/api/webhooks/..."
```

### 6. Test Run (Dry Run)

```bash
./gcp-cost-monitor -dry-run
```

Expected output:
```
===========================================
📊 GCP Cost Anomaly Detector
===========================================
Project: your-project-id
Billing: billing_data.gcp_billing_export_v1_XXXXXX
Threshold: 20.0%
Dry Run: true
===========================================

📊 REPORT SUMMARY
-----------------------------------------
Date: 2024-03-03 10:30:00
Anomaly: false

💰 DAILY COST:
  Today:     $306.45
  Yesterday: $280.32
  Change:    9.3% 📈
  Status:    ✅ NORMAL
...
```

### 7. Setup Cron Job (Production)

Edit crontab:
```bash
crontab -e
```

Add this line to run at 8 AM every day:

```cron
0 8 * * * cd /path/to/gcp-cost-monitor && ./gcp-cost-monitor >> /var/log/gcp-cost-monitor.log 2>&1
```

For different timezone, adjust accordingly. Example for Asia/Ho_Chi_Minh (UTC+7):
```cron
0 8 * * * TZ='Asia/Ho_Chi_Minh' cd /path/to/gcp-cost-monitor && ./gcp-cost-monitor
```

## Usage

### Command-line Options

```bash
./gcp-cost-monitor \
  -project=YOUR_PROJECT_ID \
  -discord=YOUR_WEBHOOK_URL \
  -dataset=billing_data \
  -table=gcp_billing_export_v1_XXXXXX \
  -threshold=20 \
  -dry-run=false
```

**Flags:**
- `-project` : GCP Project ID (env: `GCP_PROJECT_ID`)
- `-discord` : Discord webhook URL (env: `DISCORD_WEBHOOK`)
- `-dataset` : BigQuery billing dataset (default: `billing_data`)
- `-table` : BigQuery billing table name (required)
- `-threshold` : Anomaly threshold % (default: `20`)
- `-dry-run` : Run without sending to Discord (useful for testing)

### Environment Variables

```bash
GCP_PROJECT_ID          # Your GCP project ID
GOOGLE_APPLICATION_CREDENTIALS  # Path to service account key
BILLING_TABLE           # BigQuery billing table name
DISCORD_WEBHOOK         # Discord webhook URL
```

## Report Format

The Discord report includes:

### 1. 📊 Cost Overview
- Current/previous day costs
- Percentage change
- Anomaly status (✅/⚠️/🚨)

### 2. 🔧 Service Breakdown
- Individual service costs
- Change percentage
- Anomaly flags

### 3. 🔍 BigQuery Deep Dive (if anomaly detected)
- Top expensive jobs
- Job patterns (frequent queries)
- Cost by user
- Cost by query type

### 4. 💡 Recommendations
- Actionable insights
- Cost optimization suggestions

## Troubleshooting

### "BigQuery client creation failed"
- Check `GOOGLE_APPLICATION_CREDENTIALS` points to valid key
- Verify service account has BigQuery permissions

### "Failed to execute query"
- Verify `BILLING_TABLE` name is correct
- Check billing export is configured in GCP Console
- Ensure data has arrived (can take 1-2 days after setup)

### "Webhook returned 401"
- Check Discord webhook URL is valid
- Verify channel exists and webhook has access

### No data showing
- Billing export needs 1-2 days to populate initial data
- Check that queries return results with test queries in BigQuery

## Monitoring

### View Logs

```bash
# Last 50 lines
tail -50 /var/log/gcp-cost-monitor.log

# Real-time
tail -f /var/log/gcp-cost-monitor.log
```

### Manual Execution

```bash
# Run manually anytime
cd /path/to/gcp-cost-monitor && ./gcp-cost-monitor

# With specific threshold
./gcp-cost-monitor -threshold=30
```

## Development

### Structure

```
.
├── main.go          # Entry point + CLI
├── models.go        # Data structures
├── gcp.go           # GCP API interactions
├── analyzer.go      # Cost analysis logic
├── discord.go       # Discord webhook
├── go.mod           # Dependencies
├── README.md        # This file
├── .env.example     # Environment template
└── setup.sh         # Setup script
```

### Adding New Features

1. **New GCP queries**: Add to `gcp.go`
2. **New analysis logic**: Add to `analyzer.go`
3. **New Discord format**: Update `discord.go`
4. **New data types**: Add to `models.go`

## License

MIT

## Support

Issues? Check:
1. GCP credentials are valid
2. BigQuery tables exist and have data
3. Discord webhook URL is correct
4. Service account has proper IAM roles
