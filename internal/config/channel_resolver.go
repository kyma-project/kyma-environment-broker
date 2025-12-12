package config

import (
	"fmt"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"gopkg.in/yaml.v3"
)

type ChannelResolver interface {
	GetChannelForPlan(planName string) (string, error)
}

type channelResolver struct {
	configProvider    ConfigMapConfigProvider
	logger            *slog.Logger
	planNames         []string
	planChannelsCache map[string]string
}

func NewChannelResolver(configProvider ConfigMapConfigProvider, planNames []string, logger *slog.Logger) (ChannelResolver, error) {
	resolver := &channelResolver{
		configProvider: configProvider,
		planNames:      planNames,
		logger:         logger.With("component", "ChannelResolver"),
	}

	if err := resolver.loadChannels(); err != nil {
		return nil, fmt.Errorf("while loading channels: %w", err)
	}

	return resolver, nil
}

func (r *channelResolver) GetChannelForPlan(planName string) (string, error) {
	if channel, exists := r.planChannelsCache[planName]; exists {
		return channel, nil
	}

	if defaultChannel, exists := r.planChannelsCache["default"]; exists {
		r.logger.Info(fmt.Sprintf("No channel configured for plan %s, using default channel: %s", planName, defaultChannel))
		return defaultChannel, nil
	}

	return "", fmt.Errorf("no channel configured for plan %s and no default found", planName)
}

func (r *channelResolver) loadChannels() error {
	r.logger.Info("Loading channels from runtime configuration")

	r.planChannelsCache = make(map[string]string)

	defaultChannel, err := r.loadChannelForPlan("default")
	if err != nil {
		return fmt.Errorf("failed to load default channel (required): %w", err)
	}
	r.planChannelsCache["default"] = defaultChannel
	r.logger.Info(fmt.Sprintf("Loaded default channel: %s", defaultChannel))

	for _, planName := range r.planNames {
		if planName == "default" {
			continue
		}

		channel, err := r.loadChannelForPlan(planName)
		if err != nil {
			r.logger.Info(fmt.Sprintf("Plan %s will use default channel: %s", planName, defaultChannel))
			continue
		}
		r.planChannelsCache[planName] = channel
		r.logger.Info(fmt.Sprintf("Loaded channel for plan %s: %s", planName, channel))
	}

	return nil
}

func (r *channelResolver) loadChannelForPlan(planName string) (string, error) {
	cfg := &internal.ConfigForPlan{}
	err := r.configProvider.Provide(planName, cfg)
	if err != nil {
		return "", fmt.Errorf("while getting config for plan %s: %w", planName, err)
	}

	return r.extractChannelFromKymaTemplate(cfg.KymaTemplate)
}

func (r *channelResolver) extractChannelFromKymaTemplate(kymaTemplate string) (string, error) {
	if kymaTemplate == "" {
		return "", fmt.Errorf("kyma-template is empty")
	}

	var template map[string]interface{}
	if err := yaml.Unmarshal([]byte(kymaTemplate), &template); err != nil {
		return "", fmt.Errorf("while unmarshaling kyma-template: %w", err)
	}

	if spec, ok := template["spec"].(map[string]interface{}); ok {
		if channel, ok := spec["channel"].(string); ok {
			return channel, nil
		}
	}

	return "", fmt.Errorf("channel not found in kyma-template spec")
}
