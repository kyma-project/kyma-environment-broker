package ans

import (
	"context"
	"log/slog"
)

type AnsService struct {
	ctx    context.Context
	cfg    Config
	client *RateLimitedAnsClient
}

func NewAnsService(ctx context.Context, cfg Config, logger *slog.Logger) *AnsService {
	notificationClient := NewRateLimitedAnsClient(ctx, cfg, logger.With("component", "ANS-notification-client"))
	return &AnsService{
		ctx:    ctx,
		cfg:    cfg,
		client: notificationClient,
	}
}

func (s *AnsService) PostNotification(notification Notification) error {

	s.client.log.Debug("Posting notification to ANS", "notification", notification)
	err := s.client.postNotification(notification)
	if err != nil {
		s.client.log.Error("Failed to post notification to ANS", "error", err)
		return err
	}

	s.client.log.Debug("Notification posted successfully")
	return nil
}
