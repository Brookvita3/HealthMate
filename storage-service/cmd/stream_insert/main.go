package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"api-gateway/config"
	"api-gateway/internal/metric"
	postgrePlatform "api-gateway/internal/platform/postgres"
)

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	// Kết nối database
	pool, err := postgrePlatform.NewTimeScaleDBConnFromURL(cfg.TimeScaleDbURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	defer pool.Close()
	log.Println("Successfully connected to TimescaleDB")

	// Tạo repository và service
	metricRepo := metric.NewRepository(pool)
	metricService := metric.NewMetricService(metricRepo)

	// User ID mẫu
	userID := "00000000-0000-0000-0000-000000000001"
	
	fmt.Println("==============================================")
	fmt.Println("STREAMING INSERT - 30 records/minute for each metric")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("==============================================")
	fmt.Printf("User ID: %s\n", userID)
	fmt.Println()

	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Counter
	totalInserted := 0
	startTime := time.Now()

	// Ticker để insert mỗi phút
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Insert ngay lần đầu tiên
	insertBatch(ctx, metricService, userID, &totalInserted)

	// Loop insert mỗi phút
	for {
		select {
		case <-ticker.C:
			insertBatch(ctx, metricService, userID, &totalInserted)
			
			// Hiển thị thống kê
			elapsed := time.Since(startTime)
			fmt.Printf("\n--- Statistics ---\n")
			fmt.Printf("Total batches: %d\n", totalInserted/90)
			fmt.Printf("Total records: %d\n", totalInserted)
			fmt.Printf("Running time: %s\n", elapsed.Round(time.Second))
			fmt.Printf("Average: %.2f records/sec\n", float64(totalInserted)/elapsed.Seconds())
			fmt.Println()
		}
	}
}

func insertBatch(ctx context.Context, metricService metric.Service, userID string, totalInserted *int) {
	now := time.Now()
	fmt.Printf("[%s] Inserting batch...\n", now.Format("2006-01-02 15:04:05"))

	successCount := 0
	errorCount := 0

	// Insert 30 heart_rate records
	for i := 0; i < 30; i++ {
		timestamp := now.Add(time.Duration(i) * time.Second)
		heartRate := 60.0 + rand.Float64()*40.0 // 60-100 bpm

		metric := metric.HealthMetric{
			UserID:    userID,
			Type:      "heart_rate",
			Value:     heartRate,
			Timestamp: timestamp,
		}

		if err := metricService.RecordMetric(ctx, metric); err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// Insert 30 steps_count records
	for i := 0; i < 30; i++ {
		timestamp := now.Add(time.Duration(i) * time.Second)
		steps := rand.Intn(500) + 100 // 100-600 steps

		metric := metric.HealthMetric{
			UserID:    userID,
			Type:      "steps_count",
			Value:     float64(steps),
			Timestamp: timestamp,
		}

		if err := metricService.RecordMetric(ctx, metric); err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// Insert 30 calories_burned records
	for i := 0; i < 30; i++ {
		timestamp := now.Add(time.Duration(i) * time.Second)
		calories := 1.0 + rand.Float64()*4.0 // 1-5 calories

		metric := metric.HealthMetric{
			UserID:    userID,
			Type:      "calories_burned",
			Value:     calories,
			Timestamp: timestamp,
		}

		if err := metricService.RecordMetric(ctx, metric); err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	*totalInserted += successCount

	fmt.Printf("  ✓ Inserted: %d records (errors: %d)\n", successCount, errorCount)
	fmt.Printf("  Total so far: %d records\n", *totalInserted)
}
