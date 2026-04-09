package medication

import (
	"context"
	"log"
	"time"
)

func StartScheduler(ctx context.Context, service Service) {
	go func() {
		log.Println("Medication Reminder Scheduler started.")

		// Wait for the next minute to start
		now := time.Now()
		nextMinute := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(nextMinute))

		// Ticker runs every minute
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Medication Reminder Scheduler stopping...")
				return
			case <-ticker.C:
				err := service.CheckAndTriggerReminders(ctx)
				if err != nil {
					log.Printf("Error in scheduler: %v", err)
				}
			}
		}
	}()
}
