package tasks

import (
	"context"
	"log"
	"time"

	"github.com/J0es1ick/shortli/internal/repository"
)

type MaintenanceTask struct {
	repository             *repository.MaintenanceRepository
	interval               time.Duration
	analyticsRetentionDays int
	reportRetentionDays    int
}

func NewMaintenanceTask(
	repository *repository.MaintenanceRepository,
	interval time.Duration,
	analyticsRetentionDays int,
	reportRetentionDays int,
) *MaintenanceTask {
	return &MaintenanceTask{
		repository: repository, interval: interval,
		analyticsRetentionDays: analyticsRetentionDays,
		reportRetentionDays:    reportRetentionDays,
	}
}

func (t *MaintenanceTask) Start(ctx context.Context) {
	t.run(ctx)
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (t *MaintenanceTask) run(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	result, err := t.repository.Run(
		ctx, t.analyticsRetentionDays, t.reportRetentionDays,
	)
	if err != nil {
		log.Printf("maintenance failed: %v", err)
		return
	}
	log.Printf(
		"maintenance completed expired_links=%d expired_sessions=%d old_click_events=%d old_reports=%d",
		result.ExpiredLinks, result.ExpiredSessions,
		result.OldClickEvents, result.OldReports,
	)
}
