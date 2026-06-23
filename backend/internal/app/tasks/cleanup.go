package tasks

import (
	"log"
	"time"

	"github.com/J0es1ick/shortli/internal/repository"
)

type CleanupTask struct {
	urlRepository 	*repository.UrlRepository
	interval 		time.Duration
}

func NewCleanupTask(urlRepository *repository.UrlRepository, interval time.Duration) *CleanupTask {
	return &CleanupTask{
		urlRepository: urlRepository,
		interval: interval,
	}
}

func (c *CleanupTask) Start() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		c.runCleanup()
	}
}

func (c *CleanupTask) runCleanup() {
	log.Println("Starting cleanup of old URLs...")
    
    count, err := c.urlRepository.DeleteOldUrls()
    if err != nil {
        log.Printf("Cleanup failed: %v", err)
        return
    }
    
    if count > 0 {
        log.Printf("Cleanup completed: deleted %d old URLs", count)
    } else {
        log.Println("Cleanup completed: no old URLs found")
    }
}

func (t *CleanupTask) RunOnce() (int64, error) {
    return t.urlRepository.DeleteOldUrls()
}