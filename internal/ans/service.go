package ans

import (
	"context"
	"log/slog"
)

type Service struct {
	ctx    context.Context
	cfg    Config
	client *RateLimitedAnsClient
}

func NewAnsService(ctx context.Context, cfg Config, logger *slog.Logger) *Service {
	notificationClient := NewRateLimitedAnsClient(ctx, cfg, logger.With("component", "ANS-notification-client"))
	return &Service{
		ctx:    ctx,
		cfg:    cfg,
		client: notificationClient,
	}
}

func (s *Service) PostNotification(notification Notification) error {
	if !s.cfg.Enabled {
		s.client.log.Debug("ANS notifications are disabled, skipping posting notification")
		return nil
	}

	s.client.log.Info("Posting notification", "notification", notification)
	err := s.client.postNotification(notification)
	if err != nil {
		s.client.log.Error("Failed to post notification", "error", err)
		return err
	}

	s.client.log.Info("Notification posted successfully")
	return nil
}
