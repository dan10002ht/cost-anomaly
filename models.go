package main

import "time"

// DailyCost represents daily cost breakdown
type DailyCost struct {
	Date              time.Time
	TotalCost         float64
	PreviousDayCost   float64
	PercentageChange  float64
	AnomalyLevel      string // "✅ NORMAL", "⚠️ MEDIUM", "🚨 HIGH", "🚨 CRITICAL"
	ServiceCosts      map[string]float64
	ServiceChanges    map[string]float64
}

// Service represents a GCP service
type Service struct {
	Name              string
	Cost              float64
	PreviousCost      float64
	PercentageChange  float64
	AnomalyLevel      string
}

// BigQueryJob represents a BigQuery job
type BigQueryJob struct {
	JobID           string
	UserEmail       string
	CreationTime    time.Time
	FinishedTime    time.Time
	DurationSeconds float64
	EstimatedCost   float64
	GBScanned       float64
	SlotSeconds     float64
	StatementType   string
	QuerySnippet    string
	FullQuery       string
}

// BigQueryAnalysis represents analysis of BQ jobs
type BigQueryAnalysis struct {
	TopExpensiveJobs []BigQueryJob
	JobPatterns      []JobPattern
	CostByUser       map[string]float64
	CostByType       map[string]float64
	TotalCost        float64
	TotalGB          float64
}

// JobPattern represents a recurring job pattern
type JobPattern struct {
	QueryPattern   string
	ExecutionCount int
	AvgSlotSeconds float64
	AvgDurationSec float64
	MaxDurationSec float64
	UniqueUsers    int
	TrendRatio     float64
}

// Report represents the complete report
type Report struct {
	Date              time.Time
	AnomalyDetected   bool
	AnomalyDate       time.Time
	DailyCost         DailyCost
	Services          []Service
	BigQueryAnalysis  *BigQueryAnalysis
	Recommendations   []string
}

// DiscordEmbed represents a Discord embed message
type DiscordEmbed struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Color       int                    `json:"color"`
	Fields      []DiscordEmbedField    `json:"fields"`
	Footer      DiscordEmbedFooter     `json:"footer"`
	Timestamp   string                 `json:"timestamp"`
}

// DiscordEmbedField represents a field in Discord embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// DiscordEmbedFooter represents footer in Discord embed
type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

// DiscordMessage represents a Discord webhook message
type DiscordMessage struct {
	Username string         `json:"username,omitempty"`
	Content  string         `json:"content,omitempty"`
	Embeds   []DiscordEmbed `json:"embeds,omitempty"`
}

// Config holds application configuration
type Config struct {
	GCPProjectID    string
	DiscordWebhook  string
	AnomalyThreshold float64 // 20 = 20%
	BillingDataset  string
	BillingTable    string
}
