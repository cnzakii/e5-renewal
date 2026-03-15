package scheduler

import (
	"e5-renewal/backend/models"
	"time"
)

// TimeWeight exposes timeWeight for testing.
func TimeWeight(t time.Time) float64 {
	return timeWeight(t)
}

// ComputeHealth exposes computeHealth for testing.
func (s *Scheduler) ComputeHealth(accountID uint) float64 {
	return s.computeHealth(s.ctx, accountID)
}

// CheckAuthExpiry exposes checkAuthExpiry for testing.
func (s *Scheduler) CheckAuthExpiry() {
	s.checkAuthExpiry(s.ctx)
}

// NotifyIfEnabled exposes notifyIfEnabled for testing.
func (s *Scheduler) NotifyIfEnabled(eventKey, title, message string) {
	var cfg models.NotificationConfig
	s.loadNotificationConfig(s.ctx, &cfg)
	s.notifyIfEnabled(&cfg, eventKey, title, message)
}

// RunTask exposes runTask for testing.
func (s *Scheduler) RunTask(accountID uint) {
	s.runTask(s.ctx, accountID)
}
