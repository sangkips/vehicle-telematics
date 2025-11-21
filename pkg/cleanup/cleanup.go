package cleanup

import (
	"fleet-backend/internal/repository"
	"log"
	"time"
)

type CleanupService struct {
	userRepo *repository.UserRepository
	interval time.Duration
	stopChan chan bool
}

func NewCleanupService(userRepo *repository.UserRepository, interval time.Duration) *CleanupService {
	return &CleanupService{
		userRepo: userRepo,
		interval: interval,
		stopChan: make(chan bool),
	}
}

// Start begins the cleanup service
func (s *CleanupService) Start() {
	log.Printf("Starting password reset token cleanup service (interval: %v)", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	s.cleanupExpiredTokens()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpiredTokens()
		case <-s.stopChan:
			log.Println("Stopping password reset token cleanup service")
			return
		}
	}
}

// Stop stops the cleanup service
func (s *CleanupService) Stop() {
	s.stopChan <- true
}

// cleanupExpiredTokens removes expired password reset tokens from the database
func (s *CleanupService) cleanupExpiredTokens() {
	count, err := s.userRepo.CleanupExpiredResetTokens()
	if err != nil {
		log.Printf("Error cleaning up expired reset tokens: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Cleaned up %d expired password reset tokens", count)
	}
}
