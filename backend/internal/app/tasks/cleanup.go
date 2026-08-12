package tasks

import (
	"context"
	"log"
	"time"

	"github.com/J0es1ick/shortli/internal/repository"
)

type CleanupTask struct {
	urlRepository *repository.UrlRepository
	interval      time.Duration
}

func NewCleanupTask(urlRepository *repository.UrlRepository, interval time.Duration) *CleanupTask {
	return &CleanupTask{
		urlRepository: urlRepository,
		interval:      interval,
	}
}

func (c *CleanupTask) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.runCleanup()
		case <-ctx.Done():
			return
		}
	}
}

func (c *CleanupTask) runCleanup() {
	log.Println("Checking expired URLs...")

	count, err := c.urlRepository.DeactivateExpiredUrls()
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Expiration check completed: paused %d URLs", count)
	} else {
		log.Println("Expiration check completed: no URLs to pause")
	}
}

func (t *CleanupTask) RunOnce() (int64, error) {
	return t.urlRepository.DeactivateExpiredUrls()
}
