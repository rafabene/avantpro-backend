package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rafabene/avantpro-backend/internal/repositories"
)

// Worker manages periodic maintenance tasks for the application.
// It runs background jobs that perform system cleanup and maintenance operations.
type Worker struct {
	orgRepo repositories.OrganizationRepositoryInterface
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWorker creates a new worker instance with the provided dependencies.
// Parameters:
//   - orgRepo: Repository interface for organization operations
//
// Returns:
//   - *Worker: Configured worker ready to start background tasks
func NewWorker(orgRepo repositories.OrganizationRepositoryInterface) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		orgRepo: orgRepo,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins running all periodic maintenance tasks.
// This method starts goroutines for each maintenance task and returns immediately.
// The tasks will continue running until Stop() is called.
func (w *Worker) Start() {
	log.Println("🔧 Starting maintenance worker...")

	// Start invitation expiry task (runs every hour)
	go w.runPeriodicTask("Expire Invitations", 1*time.Hour, w.expireInvitations)

	// Future tasks can be added here:
	// go w.runPeriodicTask("Clean Old Logs", 24*time.Hour, w.cleanOldLogs)
	// go w.runPeriodicTask("Update Statistics", 6*time.Hour, w.updateStatistics)

	log.Println("✅ Maintenance worker started successfully")
}

// Stop gracefully stops all running maintenance tasks.
// This method cancels the context which signals all goroutines to stop.
func (w *Worker) Stop() {
	log.Println("🛑 Stopping maintenance worker...")
	w.cancel()
	log.Println("✅ Maintenance worker stopped")
}

// runPeriodicTask executes a maintenance task at regular intervals.
// Parameters:
//   - taskName: Human-readable name for logging purposes
//   - interval: How often to run the task
//   - taskFunc: The function to execute
func (w *Worker) runPeriodicTask(taskName string, interval time.Duration, taskFunc func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run the task immediately on startup
	if err := taskFunc(); err != nil {
		log.Printf("❌ Error in %s: %v", taskName, err)
	} else {
		log.Printf("✅ %s completed successfully", taskName)
	}

	for {
		select {
		case <-w.ctx.Done():
			log.Printf("🔄 Stopping %s task", taskName)
			return
		case <-ticker.C:
			if err := taskFunc(); err != nil {
				log.Printf("❌ Error in %s: %v", taskName, err)
			} else {
				log.Printf("✅ %s completed successfully", taskName)
			}
		}
	}
}

// expireInvitations marks expired organization invitations as expired.
// This task runs periodically to clean up stale invitation tokens.
func (w *Worker) expireInvitations() error {
	start := time.Now()
	defer func() {
		log.Printf("⏱️  Expire invitations task took %v", time.Since(start))
	}()

	err := w.orgRepo.ExpireInvites()
	if err != nil {
		return fmt.Errorf("failed to expire invitations: %w", err)
	}

	return nil
}

// Future maintenance tasks can be added here:

// cleanOldLogs removes old log files to free up disk space
// func (w *Worker) cleanOldLogs() error {
//     // Implementation for cleaning old logs
//     return nil
// }

// updateStatistics refreshes cached statistics and metrics
// func (w *Worker) updateStatistics() error {
//     // Implementation for updating statistics
//     return nil
// }
