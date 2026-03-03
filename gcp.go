package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// GetDailyCostBreakdown fetches daily cost by service
func GetDailyCostBreakdown(ctx context.Context, config Config, days int) (map[string]interface{}, error) {
	client, err := bigquery.NewClient(ctx, config.GCPProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	query := fmt.Sprintf(`
	WITH daily_service_cost AS (
	  SELECT 
	    DATE(usage_start_time) as usage_date,
	    service.description as service_name,
	    ROUND(SUM(cost), 2) as daily_cost
	  FROM %s.%s
	  WHERE DATE(usage_start_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
	    AND DATE(usage_start_time) <= CURRENT_DATE()
	  GROUP BY usage_date, service_name
	)
	SELECT 
	  usage_date,
	  service_name,
	  daily_cost,
	  LAG(daily_cost) OVER (PARTITION BY service_name ORDER BY usage_date) as prev_day_cost
	FROM daily_service_cost
	ORDER BY usage_date DESC, service_name
	`, config.BillingDataset, config.BillingTable, days)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	result := make(map[string]interface{})
	result["services"] = make(map[string]map[string]interface{})

	for {
		var row struct {
			UsageDate    time.Time
			ServiceName  string
			DailyCost    float64
			PrevDayCost  *float64
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		if _, ok := result["services"].(map[string]map[string]interface{})[row.ServiceName]; !ok {
			result["services"].(map[string]map[string]interface{})[row.ServiceName] = make(map[string]interface{})
		}

		svc := result["services"].(map[string]map[string]interface{})[row.ServiceName]
		svc["date"] = row.UsageDate
		svc["cost"] = row.DailyCost
		if row.PrevDayCost != nil {
			svc["prev_cost"] = *row.PrevDayCost
			pct := ((row.DailyCost - *row.PrevDayCost) / *row.PrevDayCost) * 100
			svc["pct_change"] = math.Round(pct*100) / 100
		}
	}

	return result, nil
}

// GetTotalDailyCost fetches total daily cost for last N days
func GetTotalDailyCost(ctx context.Context, config Config, days int) (map[time.Time]float64, error) {
	client, err := bigquery.NewClient(ctx, config.GCPProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	query := fmt.Sprintf(`
	SELECT 
	  DATE(usage_start_time) as usage_date,
	  ROUND(SUM(cost), 2) as total_cost
	FROM %s.%s
	WHERE DATE(usage_start_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
	GROUP BY usage_date
	ORDER BY usage_date DESC
	`, config.BillingDataset, config.BillingTable, days)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	costs := make(map[time.Time]float64)

	for {
		var row struct {
			UsageDate time.Time
			TotalCost float64
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		costs[row.UsageDate] = row.TotalCost
	}

	return costs, nil
}

// GetExpensiveBigQueryJobs fetches most expensive BQ jobs from a specific date
func GetExpensiveBigQueryJobs(ctx context.Context, config Config, targetDate time.Time, limit int) ([]BigQueryJob, error) {
	client, err := bigquery.NewClient(ctx, config.GCPProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	query := fmt.Sprintf(`
	SELECT 
	  job_id,
	  user_email,
	  creation_time,
	  job_finished_time,
	  ROUND(TIMESTAMP_DIFF(job_finished_time, creation_time, SECOND), 2) as duration_sec,
	  ROUND(total_bytes_processed / 1e12 * 6.25, 4) as estimated_cost_usd,
	  ROUND(total_bytes_processed / 1e9, 2) as gb_scanned,
	  ROUND(total_slot_ms / 1000, 2) as slot_seconds,
	  statement_type,
	  SUBSTR(query, 1, 300) as query_snippet,
	  query
	FROM `region-us`.INFORMATION_SCHEMA.JOBS_BY_PROJECT
	WHERE DATE(creation_time) = '%s'
	  AND state = 'DONE'
	  AND query NOT LIKE '%%INFORMATION_SCHEMA%%'
	ORDER BY estimated_cost_usd DESC
	LIMIT %d
	`, targetDate.Format("2006-01-02"), limit)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	var jobs []BigQueryJob

	for {
		var row struct {
			JobID           string
			UserEmail       string
			CreationTime    time.Time
			JobFinishedTime time.Time
			DurationSec     float64
			EstimatedCost   float64
			GBScanned       float64
			SlotSeconds     float64
			StatementType   string
			QuerySnippet    string
			Query           string
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		jobs = append(jobs, BigQueryJob{
			JobID:           row.JobID,
			UserEmail:       row.UserEmail,
			CreationTime:    row.CreationTime,
			FinishedTime:    row.JobFinishedTime,
			DurationSeconds: row.DurationSec,
			EstimatedCost:   row.EstimatedCost,
			GBScanned:       row.GBScanned,
			SlotSeconds:     row.SlotSeconds,
			StatementType:   row.StatementType,
			QuerySnippet:    row.QuerySnippet,
			FullQuery:       row.Query,
		})
	}

	return jobs, nil
}

// GetBigQueryJobPatterns fetches frequently executed job patterns
func GetBigQueryJobPatterns(ctx context.Context, config Config, days int) ([]JobPattern, error) {
	client, err := bigquery.NewClient(ctx, config.GCPProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	query := fmt.Sprintf(`
	SELECT 
	  SUBSTR(query, 1, 150) as query_pattern,
	  COUNT(*) as execution_count,
	  ROUND(AVG(total_slot_ms / 1000), 2) as avg_slot_sec,
	  ROUND(AVG(TIMESTAMP_DIFF(job_finished_time, creation_time, SECOND)), 2) as avg_duration_sec,
	  ROUND(MAX(TIMESTAMP_DIFF(job_finished_time, creation_time, SECOND)), 2) as max_duration_sec,
	  COUNT(DISTINCT user_email) as unique_users
	FROM `region-us`.INFORMATION_SCHEMA.JOBS_BY_PROJECT
	WHERE creation_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL %d DAY)
	  AND state = 'DONE'
	  AND statement_type IN ('SELECT', 'INSERT', 'UPDATE', 'DELETE')
	  AND query NOT LIKE '%%INFORMATION_SCHEMA%%'
	GROUP BY query_pattern
	HAVING execution_count > 3
	ORDER BY execution_count DESC
	LIMIT 30
	`, days)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	var patterns []JobPattern

	for {
		var row struct {
			QueryPattern   string
			ExecutionCount int64
			AvgSlotSec     float64
			AvgDurationSec float64
			MaxDurationSec float64
			UniqueUsers    int64
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		patterns = append(patterns, JobPattern{
			QueryPattern:   row.QueryPattern,
			ExecutionCount: int(row.ExecutionCount),
			AvgSlotSeconds: row.AvgSlotSec,
			AvgDurationSec: row.AvgDurationSec,
			MaxDurationSec: row.MaxDurationSec,
			UniqueUsers:    int(row.UniqueUsers),
		})
	}

	return patterns, nil
}

// GetBigQueryCostByUser fetches cost breakdown by user
func GetBigQueryCostByUser(ctx context.Context, config Config, targetDate time.Time) (map[string]float64, error) {
	client, err := bigquery.NewClient(ctx, config.GCPProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	query := fmt.Sprintf(`
	SELECT 
	  user_email,
	  SUM(ROUND(total_bytes_processed / 1e12 * 6.25, 4)) as total_cost
	FROM `region-us`.INFORMATION_SCHEMA.JOBS_BY_PROJECT
	WHERE DATE(creation_time) = '%s'
	  AND state = 'DONE'
	GROUP BY user_email
	ORDER BY total_cost DESC
	`, targetDate.Format("2006-01-02"))

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	costByUser := make(map[string]float64)

	for {
		var row struct {
			UserEmail string
			TotalCost float64
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		costByUser[row.UserEmail] = row.TotalCost
	}

	return costByUser, nil
}

// GetBigQueryCostByType fetches cost breakdown by query type
func GetBigQueryCostByType(ctx context.Context, config Config, targetDate time.Time) (map[string]float64, error) {
	client, err := bigquery.NewClient(ctx, config.GCPProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	query := fmt.Sprintf(`
	SELECT 
	  statement_type,
	  SUM(ROUND(total_bytes_processed / 1e12 * 6.25, 4)) as total_cost
	FROM `region-us`.INFORMATION_SCHEMA.JOBS_BY_PROJECT
	WHERE DATE(creation_time) = '%s'
	  AND state = 'DONE'
	GROUP BY statement_type
	ORDER BY total_cost DESC
	`, targetDate.Format("2006-01-02"))

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	costByType := make(map[string]float64)

	for {
		var row struct {
			StatementType string
			TotalCost     float64
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		costByType[row.StatementType] = row.TotalCost
	}

	return costByType, nil
}

func init() {
	log.Println("GCP module initialized")
}
