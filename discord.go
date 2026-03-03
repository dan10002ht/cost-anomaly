package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SendDiscordReport sends the report to Discord webhook
func SendDiscordReport(webhookURL string, report *Report) error {
	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL is empty")
	}

	// Build the message
	message := buildDiscordMessage(report)

	// Marshal to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send to Discord
	resp, err := http.Post(
		webhookURL,
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ Report sent to Discord successfully")
	return nil
}

// buildDiscordMessage builds a Discord embed message
func buildDiscordMessage(report *Report) DiscordMessage {
	message := DiscordMessage{
		Username: "📊 GCP Cost Monitor",
	}

	// Main embed
	mainEmbed := buildMainEmbed(report)
	message.Embeds = append(message.Embeds, mainEmbed)

	// Service breakdown embed
	if len(report.Services) > 0 {
		serviceEmbed := buildServiceEmbed(report.Services)
		message.Embeds = append(message.Embeds, serviceEmbed)
	}

	// BigQuery details embed
	if report.BigQueryAnalysis != nil {
		bqEmbed := buildBigQueryEmbed(report.BigQueryAnalysis)
		message.Embeds = append(message.Embeds, bqEmbed)
	}

	// Recommendations embed
	if len(report.Recommendations) > 0 {
		recEmbed := buildRecommendationsEmbed(report.Recommendations)
		message.Embeds = append(message.Embeds, recEmbed)
	}

	return message
}

// buildMainEmbed builds the main cost overview embed
func buildMainEmbed(report *Report) DiscordEmbed {
	embed := DiscordEmbed{
		Title:     "📊 GCP Daily Cost Report",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Footer: DiscordEmbedFooter{
			Text: "GCP Cost Monitor",
		},
	}

	// Color based on anomaly level
	switch report.DailyCost.AnomalyLevel {
	case "🚨 CRITICAL":
		embed.Color = 16711680 // Red
	case "🚨 HIGH":
		embed.Color = 16744448 // Orange-Red
	case "⚠️ MEDIUM":
		embed.Color = 16776960 // Orange-Yellow
	default:
		embed.Color = 65280 // Green
	}

	// Status
	statusText := "✅ NORMAL"
	if report.AnomalyDetected {
		statusText = report.DailyCost.AnomalyLevel
	}
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Status",
		Value:  statusText,
		Inline: true,
	})

	// Date
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Date",
		Value:  report.DailyCost.Date.Format("2006-01-02"),
		Inline: true,
	})

	// Today's cost
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Today Cost",
		Value:  fmt.Sprintf("$%.2f", report.DailyCost.TotalCost),
		Inline: true,
	})

	// Yesterday's cost
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Yesterday Cost",
		Value:  fmt.Sprintf("$%.2f", report.DailyCost.PreviousDayCost),
		Inline: true,
	})

	// Change
	changeIcon := "📈"
	if report.DailyCost.PercentageChange < 0 {
		changeIcon = "📉"
	}

	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Change",
		Value:  fmt.Sprintf("%s %.1f%%", changeIcon, report.DailyCost.PercentageChange),
		Inline: true,
	})

	// Difference
	diff := report.DailyCost.TotalCost - report.DailyCost.PreviousDayCost
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Difference",
		Value:  fmt.Sprintf("$%.2f", diff),
		Inline: true,
	})

	return embed
}

// buildServiceEmbed builds service-level breakdown embed
func buildServiceEmbed(services []Service) DiscordEmbed {
	embed := DiscordEmbed{
		Title:     "🔧 Service Breakdown",
		Color:     3447003, // Blue
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	serviceText := ""
	for _, service := range services {
		// Truncate long service names
		name := service.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		pctChange := fmt.Sprintf("%.1f%%", service.PercentageChange)
		changeIcon := "📈"
		if service.PercentageChange < 0 {
			changeIcon = "📉"
		}

		serviceText += fmt.Sprintf(
			"%s **%s**: $%.2f (%s %s)\n",
			service.AnomalyLevel,
			name,
			service.Cost,
			changeIcon,
			pctChange,
		)
	}

	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:  "Services",
		Value: serviceText,
	})

	return embed
}

// buildBigQueryEmbed builds BigQuery analysis embed
func buildBigQueryEmbed(analysis *BigQueryAnalysis) DiscordEmbed {
	embed := DiscordEmbed{
		Title:     "🔍 BigQuery Analysis",
		Color:     9442302, // Purple
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Total stats
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Total Cost",
		Value:  fmt.Sprintf("$%.2f", analysis.TotalCost),
		Inline: true,
	})

	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Total GB Scanned",
		Value:  fmt.Sprintf("%.2f GB", analysis.TotalGB),
		Inline: true,
	})

	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Expensive Jobs",
		Value:  fmt.Sprintf("%d", len(analysis.TopExpensiveJobs)),
		Inline: true,
	})

	// Top expensive jobs
	if len(analysis.TopExpensiveJobs) > 0 {
		jobsText := ""
		for i, job := range analysis.TopExpensiveJobs {
			if i >= 5 { // Limit to top 5
				break
			}

			jobsText += fmt.Sprintf(
				"`%s` - $%.4f (%.2f GB, %ds)\n",
				job.JobID[:16],
				job.EstimatedCost,
				job.GBScanned,
				int(job.DurationSeconds),
			)
		}

		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:  "Top 5 Expensive Jobs",
			Value: jobsText,
		})
	}

	// Cost by user
	if len(analysis.CostByUser) > 0 {
		userText := ""
		count := 0
		for user, cost := range analysis.CostByUser {
			if count >= 3 { // Limit to top 3
				break
			}
			userText += fmt.Sprintf("• **%s**: $%.2f\n", user, cost)
			count++
		}

		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:  "Cost by User",
			Value: userText,
		})
	}

	// Job patterns
	if len(analysis.JobPatterns) > 0 {
		patternsText := ""
		for i, pattern := range analysis.JobPatterns {
			if i >= 3 { // Limit to top 3
				break
			}

			patternsText += fmt.Sprintf(
				"• **%d** executions - Avg: %ds, Slots: %.2f\n",
				pattern.ExecutionCount,
				int(pattern.AvgDurationSec),
				pattern.AvgSlotSeconds,
			)
		}

		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:  "Top Job Patterns",
			Value: patternsText,
		})
	}

	return embed
}

// buildRecommendationsEmbed builds recommendations embed
func buildRecommendationsEmbed(recommendations []string) DiscordEmbed {
	embed := DiscordEmbed{
		Title:     "💡 Recommendations",
		Color:     10181046, // Yellow
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	recText := ""
	for _, rec := range recommendations {
		recText += fmt.Sprintf("• %s\n", rec)
	}

	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:  "Actions",
		Value: recText,
	})

	return embed
}
