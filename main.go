package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Parse command-line flags
	projectID := flag.String("project", os.Getenv("GCP_PROJECT_ID"), "GCP Project ID")
	discordWebhook := flag.String("discord", os.Getenv("DISCORD_WEBHOOK"), "Discord Webhook URL")
	billingDataset := flag.String("dataset", os.Getenv("BILLING_DATASET"), "Billing dataset name (default: billing_data)")
	billingTable := flag.String("table", os.Getenv("BILLING_TABLE"), "Billing table name")
	threshold := flag.Float64("threshold", 20.0, "Anomaly threshold percentage (default: 20)")
	dryRun := flag.Bool("dry-run", false, "Dry run mode (don't send to Discord)")

	flag.Parse()

	// Validate inputs
	if *projectID == "" {
		log.Fatal("❌ GCP_PROJECT_ID not set. Use -project or set GCP_PROJECT_ID env var")
	}

	if *discordWebhook == "" && !*dryRun {
		log.Fatal("❌ DISCORD_WEBHOOK not set. Use -discord or set DISCORD_WEBHOOK env var (or use -dry-run)")
	}

	if *billingDataset == "" {
		*billingDataset = "billing_data"
	}

	if *billingTable == "" {
		log.Fatal("❌ BILLING_TABLE not set. Use -table or set BILLING_TABLE env var")
	}

	// Create config
	config := Config{
		GCPProjectID:     *projectID,
		DiscordWebhook:   *discordWebhook,
		AnomalyThreshold: *threshold,
		BillingDataset:   *billingDataset,
		BillingTable:     *billingTable,
	}

	// Run analysis
	log.Println("===========================================")
	log.Println("📊 GCP Cost Anomaly Detector")
	log.Println("===========================================")
	log.Printf("Project: %s\n", config.GCPProjectID)
	log.Printf("Billing: %s.%s\n", config.BillingDataset, config.BillingTable)
	log.Printf("Threshold: %.1f%%\n", config.AnomalyThreshold)
	log.Printf("Dry Run: %v\n", *dryRun)
	log.Println("===========================================")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Analyze costs
	report, err := AnalyzeCostAnomalies(ctx, config)
	if err != nil {
		log.Fatalf("❌ Analysis failed: %v", err)
	}

	// Print report summary
	printReportSummary(report)

	// Send to Discord if not dry run
	if !*dryRun {
		if err := SendDiscordReport(config.DiscordWebhook, report); err != nil {
			log.Fatalf("❌ Failed to send Discord report: %v", err)
		}
	} else {
		log.Println("ℹ️  Dry run mode - skipping Discord send")
	}

	log.Println("===========================================")
	log.Println("✅ Analysis completed successfully!")
	log.Println("===========================================")
}

// printReportSummary prints report summary to console
func printReportSummary(report *Report) {
	fmt.Println()
	fmt.Println("📊 REPORT SUMMARY")
	fmt.Println("-----------------------------------------")
	fmt.Printf("Date: %s\n", report.Date.Format("2006-01-02 15:04:05"))
	fmt.Printf("Anomaly: %v\n", report.AnomalyDetected)
	fmt.Println()

	fmt.Println("💰 DAILY COST:")
	fmt.Printf("  Today:     $%.2f\n", report.DailyCost.TotalCost)
	fmt.Printf("  Yesterday: $%.2f\n", report.DailyCost.PreviousDayCost)
	fmt.Printf("  Change:    %.1f%% %s\n", report.DailyCost.PercentageChange, getChangeIcon(report.DailyCost.PercentageChange))
	fmt.Printf("  Status:    %s\n", report.DailyCost.AnomalyLevel)
	fmt.Println()

	if len(report.Services) > 0 {
		fmt.Println("🔧 SERVICES:")
		for _, svc := range report.Services {
			fmt.Printf("  %s %s: $%.2f (%.1f%%)\n", svc.AnomalyLevel, svc.Name, svc.Cost, svc.PercentageChange)
		}
		fmt.Println()
	}

	if report.BigQueryAnalysis != nil {
		fmt.Println("🔍 BIGQUERY ANALYSIS:")
		fmt.Printf("  Total Cost: $%.2f\n", report.BigQueryAnalysis.TotalCost)
		fmt.Printf("  GB Scanned: %.2f\n", report.BigQueryAnalysis.TotalGB)
		fmt.Printf("  Expensive Jobs: %d\n", len(report.BigQueryAnalysis.TopExpensiveJobs))

		if len(report.BigQueryAnalysis.TopExpensiveJobs) > 0 {
			fmt.Println("  Top 3 Jobs:")
			for i, job := range report.BigQueryAnalysis.TopExpensiveJobs {
				if i >= 3 {
					break
				}
				fmt.Printf("    %d. [%s] User: %s, Cost: $%.4f, GB: %.2f\n",
					i+1, job.JobID[:16], job.UserEmail, job.EstimatedCost, job.GBScanned)
			}
		}
		fmt.Println()
	}

	if len(report.Recommendations) > 0 {
		fmt.Println("💡 RECOMMENDATIONS:")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
		fmt.Println()
	}

	fmt.Println("-----------------------------------------")
}

// getChangeIcon returns an icon based on change percentage
func getChangeIcon(change float64) string {
	if change > 0 {
		return "📈"
	} else if change < 0 {
		return "📉"
	}
	return "➡️"
}
