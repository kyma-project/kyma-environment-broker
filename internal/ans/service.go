package ans

import (
	"context"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal/ans/notifications"
)

type Service struct {
	ctx                context.Context
	cfg                Config
	notificationClient *RateLimitedNotificationClient
}

func NewAnsService(ctx context.Context, cfg Config, logger *slog.Logger) *Service {
	notificationClient := NewNotificationClient(ctx, cfg, logger.With("component", "ANS-notification-notificationClient"))
	return &Service{
		ctx:                ctx,
		cfg:                cfg,
		notificationClient: notificationClient,
	}
}

func (s *Service) PostNotification(notification notifications.Notification) error {
	if !s.cfg.Enabled {
		s.notificationClient.log.Debug("ANS notifications are disabled, skipping posting notification")
		return nil
	}

	s.notificationClient.log.Info("Posting notification", "notification", notification)
	err := s.notificationClient.postNotification(notification)
	if err != nil {
		s.notificationClient.log.Error("Failed to post notification", "error", err)
		return err
	}

	s.notificationClient.log.Info("Notification posted successfully")
	return nil
}
