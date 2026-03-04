# GCP Cost Anomaly Monitor - Requirements Checklist

## 🎯 Pre-requisites

Trước khi chạy, anh cần chuẩn bị những thứ sau:

---

## 1️⃣ **GCP Project Setup**

### 1.1 GCP Project ID
- [ ] Có GCP project ID (ví dụ: `my-project-123`)
- [ ] Xác nhận project trong: https://console.cloud.google.com/

### 1.2 Enable BigQuery
- [ ] BigQuery API enabled
- [ ] Check: https://console.cloud.google.com/apis/api/bigquery.googleapis.com

---

## 2️⃣ **Billing Export to BigQuery**

### 2.1 Setup Billing Export
- [ ] Vào: Cloud Console → Billing → Billing Export
- [ ] Click "Export to BigQuery"
- [ ] Select project
- [ ] Dataset name: `billing_data` (hoặc custom)
- [ ] **Note down:** Exact table name (format: `gcp_billing_export_v1_XXXXXX`)
- [ ] ⚠️ **Important:** Mất 1-2 ngày để billing data populate

### 2.2 Verify Table Exists
- [ ] BigQuery → billing_data dataset
- [ ] Tìm table `gcp_billing_export_v1_...`
- [ ] Run test query:
  ```sql
  SELECT * FROM billing_data.gcp_billing_export_v1_XXXXXX LIMIT 1
  ```

---

## 3️⃣ **Service Account Setup**

### 3.1 Create Service Account
```bash
# Set your project ID
export PROJECT_ID=your-project-id

# Create service account
gcloud iam service-accounts create gcp-cost-monitor \
  --display-name="GCP Cost Monitor"
```
- [ ] Service account created

### 3.2 Grant BigQuery Permissions
```bash
# Grant BigQuery Data Viewer
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:gcp-cost-monitor@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/bigquery.dataViewer

# Grant BigQuery Job User
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:gcp-cost-monitor@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/bigquery.jobUser
```
- [ ] Permissions granted

### 3.3 Create Service Account Key
```bash
gcloud iam service-accounts keys create key.json \
  --iam-account=gcp-cost-monitor@$PROJECT_ID.iam.gserviceaccount.com
```
- [ ] `key.json` file created
- [ ] **Keep this file safe!** (Don't commit to GitHub)

---

## 4️⃣ **Discord Setup**

### 4.1 Create Discord Webhook
- [ ] Go to Discord server
- [ ] Server Settings → Integrations → Webhooks
- [ ] Click "New Webhook"
- [ ] Name: `GCP Cost Monitor`
- [ ] **Select Channel:** `#1478341525703229532` (or your preferred channel)
- [ ] Copy Webhook URL
  - Format: `https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN`

### 4.2 Test Webhook (Optional)
```bash
curl -X POST https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN \
  -H "Content-Type: application/json" \
  -d '{"content":"Test message"}'
```
- [ ] Webhook responds successfully

---

## 5️⃣ **Local Setup**

### 5.1 Install Go
- [ ] Go 1.21+ installed
  ```bash
  go version
  ```

### 5.2 Clone/Navigate Project
```bash
# Navigate to project
cd /path/to/cost-anomaly
```
- [ ] In project directory

### 5.3 Configure .env
```bash
# Copy template
cp .env.example .env

# Edit with your values
nano .env
```

**Fill these values:**
```env
GCP_PROJECT_ID=your-project-id
GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
BILLING_TABLE=gcp_billing_export_v1_XXXXXX
DISCORD_WEBHOOK=https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN
```

- [ ] GCP_PROJECT_ID set
- [ ] GOOGLE_APPLICATION_CREDENTIALS set (path to key.json)
- [ ] BILLING_TABLE set (exact table name)
- [ ] DISCORD_WEBHOOK set

### 5.4 Build Binary
```bash
go mod download
go build -o gcp-cost-monitor main.go gcp.go analyzer.go discord.go models.go
```
- [ ] Binary compiled successfully
- [ ] File `gcp-cost-monitor` exists

### 5.5 Test Dry Run
```bash
./gcp-cost-monitor -dry-run
```
- [ ] Command runs without errors
- [ ] Shows cost data and analysis
- [ ] Does NOT send to Discord

### 5.6 Test Real Send
```bash
./gcp-cost-monitor
```
- [ ] Check Discord channel for message
- [ ] Report shows cost data

---

## 6️⃣ **Schedule Cron Job** (Production)

### 6.1 Edit Crontab
```bash
crontab -e
```

### 6.2 Add Schedule
```cron
# Run daily at 8 AM
0 8 * * * cd /path/to/cost-anomaly && ./gcp-cost-monitor >> /var/log/gcp-cost-monitor.log 2>&1
```

**For different timezone:**
```cron
# Asia/Ho_Chi_Minh (UTC+7)
0 8 * * * TZ='Asia/Ho_Chi_Minh' cd /path/to/cost-anomaly && ./gcp-cost-monitor
```

- [ ] Cron job added
- [ ] Verify: `crontab -l` shows your job

---

## 7️⃣ **Monitoring**

### 7.1 View Logs
```bash
tail -f /var/log/gcp-cost-monitor.log
```
- [ ] Log file location identified

### 7.2 Manual Test
```bash
# Run manually anytime
./gcp-cost-monitor

# With custom threshold
./gcp-cost-monitor -threshold=30
```
- [ ] Can run manually anytime

---

## 📋 Summary Checklist

### GCP (Must Have)
- [ ] GCP Project ID
- [ ] BigQuery enabled
- [ ] Billing export configured
- [ ] Billing table name known
- [ ] Service account created
- [ ] Service account key (key.json)

### Discord (Must Have)
- [ ] Discord webhook URL

### Local (Must Have)
- [ ] Go 1.21+ installed
- [ ] Project cloned/available
- [ ] .env configured
- [ ] Binary compiled
- [ ] Dry run tested

### Production (Nice to Have)
- [ ] Cron job scheduled
- [ ] Logs configured
- [ ] Discord webhook tested

---

## ⚠️ Common Issues

| Issue | Solution |
|-------|----------|
| "BigQuery client creation failed" | Check GOOGLE_APPLICATION_CREDENTIALS path and key.json permissions |
| "Failed to execute query" | Verify BILLING_TABLE is correct and has data (wait 1-2 days after setup) |
| "Webhook returned 401" | Check DISCORD_WEBHOOK URL is correct |
| "No data showing" | Ensure billing export is configured and data has been exported |

---

## 🚀 Ready to Go!

Once all checkboxes are done, anh ready chạy production! 🎉

**Next step:** Tell mình when ready, mình sẽ verify setup!
