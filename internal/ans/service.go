package ans

import (
	"context"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal/ans/events"
	"github.com/kyma-project/kyma-environment-broker/internal/ans/notifications"
)

const (
	notificationServicePath = "%s/odatav2/Notification.svc/Notifications"
	eventsServicePath       = "%s/cf/producer/service/v1/resource-events"
)

type Service struct {
	ctx                context.Context
	cfg                Config
	notificationClient *NotificationsClient
	eventsClient       *EventsClient
}

func NewAnsService(ctx context.Context, cfg Config, logger *slog.Logger) *Service {
	notificationsClient := NewNotificationsClient(ctx, cfg.Notifications, logger.With("component", "ANS-notificationClient"))
	eventsClient := NewEventsClient(ctx, cfg.Events, logger.With("component", "ANS-eventsClient"))
	return &Service{
		ctx:                ctx,
		cfg:                cfg,
		notificationClient: notificationsClient,
		eventsClient:       eventsClient,
	}
}

func (s *Service) PostNotification(notification notifications.Notification) error {
	if !s.cfg.Enabled {
		s.notificationClient.log.Debug("ANS integration is disabled, skipping posting notification")
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

func (s *Service) PostEvent(event events.ResourceEvent) error {
	if !s.cfg.Enabled {
		s.eventsClient.log.Debug("ANS integration is disabled, skipping posting event")
		return nil
	}

	s.eventsClient.log.Info("Posting event", "notification", event)
	err := s.eventsClient.postEvent(event)
	if err != nil {
		s.eventsClient.log.Error("Failed to post notification", "error", err)
		return err
	}

	s.eventsClient.log.Info("Notification posted successfully")
	return nil
}
