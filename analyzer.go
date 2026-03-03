package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// AnalyzeCostAnomalies detects cost anomalies
func AnalyzeCostAnomalies(ctx context.Context, config Config) (*Report, error) {
	report := &Report{
		Date: time.Now(),
	}

	// Get daily cost breakdown
	log.Println("Fetching daily cost breakdown...")
	costs, err := GetTotalDailyCost(ctx, config, 14)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily costs: %w", err)
	}

	// Get today's and yesterday's costs
	today := time.Now().AddDate(0, 0, 0)
	yesterday := today.AddDate(0, 0, -1)

	todayCost, todayExists := costs[today]
	yesterdayCost, yesterdayExists := costs[yesterday]

	if !todayExists || !yesterdayExists {
		log.Printf("Warning: Missing cost data for today or yesterday")
		return report, nil
	}

	// Calculate percentage change
	percentChange := ((todayCost - yesterdayCost) / yesterdayCost) * 100

	// Determine anomaly level
	anomalyLevel := "✅ NORMAL"
	anomalyDetected := false

	if percentChange >= 50 {
		anomalyLevel = "🚨 CRITICAL"
		anomalyDetected = true
	} else if percentChange >= config.AnomalyThreshold {
		if percentChange >= 30 {
			anomalyLevel = "🚨 HIGH"
		} else {
			anomalyLevel = "⚠️ MEDIUM"
		}
		anomalyDetected = true
	}

	report.DailyCost = DailyCost{
		Date:             today,
		TotalCost:        todayCost,
		PreviousDayCost:  yesterdayCost,
		PercentageChange: math.Round(percentChange*100) / 100,
		AnomalyLevel:     anomalyLevel,
	}

	report.AnomalyDetected = anomalyDetected
	if anomalyDetected {
		report.AnomalyDate = today
	}

	// Get service-level breakdown
	log.Println("Fetching service-level breakdown...")
	serviceData, err := GetDailyCostBreakdown(ctx, config, 7)
	if err != nil {
		log.Printf("Warning: Failed to get service breakdown: %v", err)
	} else {
		services := extractServiceCosts(serviceData)
		report.Services = services
	}

	// If anomaly detected, get BigQuery details
	if anomalyDetected {
		log.Println("Anomaly detected! Analyzing BigQuery...")
		bqAnalysis, err := analyzeBigQuery(ctx, config, report.AnomalyDate)
		if err != nil {
			log.Printf("Warning: Failed to analyze BigQuery: %v", err)
		} else {
			report.BigQueryAnalysis = bqAnalysis
		}
	}

	// Generate recommendations
	report.Recommendations = generateRecommendations(report)

	return report, nil
}

// extractServiceCosts extracts service costs from GCP data
func extractServiceCosts(data map[string]interface{}) []Service {
	var services []Service

	if svcMap, ok := data["services"].(map[string]map[string]interface{}); ok {
		for svcName, svcData := range svcMap {
			service := Service{
				Name: svcName,
			}

			if cost, ok := svcData["cost"].(float64); ok {
				service.Cost = math.Round(cost*100) / 100
			}

			if prevCost, ok := svcData["prev_cost"].(float64); ok {
				service.PreviousCost = math.Round(prevCost*100) / 100
			}

			if pctChange, ok := svcData["pct_change"].(float64); ok {
				service.PercentageChange = math.Round(pctChange*100) / 100
			}

			// Determine anomaly level
			service.AnomalyLevel = "✅"
			if service.PercentageChange >= 50 {
				service.AnomalyLevel = "🚨"
			} else if service.PercentageChange >= 20 {
				service.AnomalyLevel = "⚠️"
			}

			services = append(services, service)
		}
	}

	return services
}

// analyzeBigQuery performs detailed BigQuery analysis
func analyzeBigQuery(ctx context.Context, config Config, targetDate time.Time) (*BigQueryAnalysis, error) {
	analysis := &BigQueryAnalysis{
		CostByUser: make(map[string]float64),
		CostByType: make(map[string]float64),
	}

	// Get expensive jobs
	log.Println("Fetching expensive BigQuery jobs...")
	jobs, err := GetExpensiveBigQueryJobs(ctx, config, targetDate, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get expensive jobs: %w", err)
	}

	analysis.TopExpensiveJobs = jobs
	for _, job := range jobs {
		analysis.TotalCost += job.EstimatedCost
		analysis.TotalGB += job.GBScanned
	}

	// Get job patterns
	log.Println("Analyzing job patterns...")
	patterns, err := GetBigQueryJobPatterns(ctx, config, 7)
	if err != nil {
		log.Printf("Warning: Failed to get job patterns: %v", err)
	} else {
		analysis.JobPatterns = patterns
	}

	// Get cost by user
	log.Println("Calculating cost by user...")
	costByUser, err := GetBigQueryCostByUser(ctx, config, targetDate)
	if err != nil {
		log.Printf("Warning: Failed to get cost by user: %v", err)
	} else {
		analysis.CostByUser = costByUser
	}

	// Get cost by type
	log.Println("Calculating cost by query type...")
	costByType, err := GetBigQueryCostByType(ctx, config, targetDate)
	if err != nil {
		log.Printf("Warning: Failed to get cost by type: %v", err)
	} else {
		analysis.CostByType = costByType
	}

	return analysis, nil
}

// generateRecommendations creates actionable recommendations
func generateRecommendations(report *Report) []string {
	var recommendations []string

	// Cost-level recommendations
	if report.AnomalyDetected {
		if report.DailyCost.PercentageChange > 50 {
			recommendations = append(recommendations,
				"🚨 Cost spiked >50%! Investigate immediately - check for runaway processes or unexpected deployments")
		} else if report.DailyCost.PercentageChange > 30 {
			recommendations = append(recommendations,
				"⚠️ Cost increased significantly - review new jobs or data volumes")
		} else {
			recommendations = append(recommendations,
				"Monitor carefully - cost trend is above normal threshold")
		}
	}

	// Service-level recommendations
	for _, service := range report.Services {
		if service.PercentageChange >= 50 {
			recommendations = append(recommendations,
				fmt.Sprintf("🔥 %s spiked %d%% - review immediately", service.Name, int(service.PercentageChange)))
		}
	}

	// BigQuery recommendations
	if report.BigQueryAnalysis != nil {
		if report.BigQueryAnalysis.TotalGB > 1000 {
			recommendations = append(recommendations,
				fmt.Sprintf("📊 Scanned %.0f GB - consider partitioning, clustering, or query caching", report.BigQueryAnalysis.TotalGB))
		}

		if len(report.BigQueryAnalysis.CostByUser) > 0 {
			var topUser string
			var topCost float64
			for user, cost := range report.BigQueryAnalysis.CostByUser {
				if cost > topCost {
					topUser = user
					topCost = cost
				}
			}

			if topCost > 10 {
				recommendations = append(recommendations,
					fmt.Sprintf("👤 Contact %s - generated $%.2f (highest cost user)", topUser, topCost))
			}
		}

		if len(report.BigQueryAnalysis.JobPatterns) > 0 {
			topPattern := report.BigQueryAnalysis.JobPatterns[0]
			if topPattern.ExecutionCount > 20 {
				recommendations = append(recommendations,
					fmt.Sprintf("🔁 Query running %d times - consider caching or consolidating", topPattern.ExecutionCount))
			}
		}
	}

	// Default recommendation if no anomaly
	if len(recommendations) == 0 && !report.AnomalyDetected {
		recommendations = append(recommendations,
			"✅ Cost is stable - continue monitoring")
	}

	return recommendations
}
