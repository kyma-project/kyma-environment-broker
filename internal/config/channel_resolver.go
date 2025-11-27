package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"gopkg.in/yaml.v2"
	coreV1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultChannelFallback = "regular"
)

type ChannelResolver interface {
	GetChannelForPlan(planName string) (string, error)
	GetAllPlanChannels() (map[string]string, error)
}

type channelResolver struct {
	ctx               context.Context
	k8sClient         client.Client
	logger            *slog.Logger
	configMapName     string
	planChannelsCache map[string]string
}

func NewChannelResolver(ctx context.Context, k8sClient client.Client, logger *slog.Logger, configMapName string) (ChannelResolver, error) {
	resolver := &channelResolver{
		ctx:           ctx,
		k8sClient:     k8sClient,
		logger:        logger.With("component", "ChannelResolver"),
		configMapName: configMapName,
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

	r.logger.Warn(fmt.Sprintf("No channel configured for plan %s and no default found, using fallback: %s", planName, defaultChannelFallback))
	return defaultChannelFallback, nil
}

func (r *channelResolver) GetAllPlanChannels() (map[string]string, error) {
	if r.planChannelsCache == nil {
		if err := r.loadChannels(); err != nil {
			return nil, err
		}
	}
	return r.planChannelsCache, nil
}

func (r *channelResolver) loadChannels() error {
	r.logger.Info(fmt.Sprintf("Loading channels from ConfigMap: %s", r.configMapName))

	cfgMap, err := r.getConfigMap()
	if err != nil {
		return fmt.Errorf("while getting ConfigMap: %w", err)
	}

	r.planChannelsCache = make(map[string]string)

	for planName, configYAML := range cfgMap.Data {
		channel, err := r.extractChannelFromConfig(configYAML)
		if err != nil {
			r.logger.Warn(fmt.Sprintf("Failed to extract channel for plan %s: %v", planName, err))
			continue
		}
		if channel != "" {
			r.planChannelsCache[planName] = channel
			r.logger.Info(fmt.Sprintf("Loaded channel for plan %s: %s", planName, channel))
		}
	}

	if len(r.planChannelsCache) == 0 {
		return fmt.Errorf("no channels found in ConfigMap %s", r.configMapName)
	}

	return nil
}

func (r *channelResolver) getConfigMap() (*coreV1.ConfigMap, error) {
	cfgMap := &coreV1.ConfigMap{}
	err := r.k8sClient.Get(r.ctx, client.ObjectKey{Namespace: namespace, Name: r.configMapName}, cfgMap)
	if err != nil {
		return nil, fmt.Errorf("ConfigMap %s does not exist in %s namespace: %w", r.configMapName, namespace, err)
	}
	return cfgMap, nil
}

func (r *channelResolver) extractChannelFromConfig(configYAML string) (string, error) {
	var config internal.ConfigForPlan
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return "", fmt.Errorf("while unmarshaling config: %w", err)
	}

	if config.KymaTemplate == "" {
		return "", fmt.Errorf("kyma-template is empty")
	}

	var kymaTemplate map[string]interface{}
	if err := yaml.Unmarshal([]byte(config.KymaTemplate), &kymaTemplate); err != nil {
		return "", fmt.Errorf("while unmarshaling kyma-template: %w", err)
	}

	if spec, ok := kymaTemplate["spec"].(map[interface{}]interface{}); ok {
		if channel, ok := spec["channel"].(string); ok {
			return channel, nil
		}
	}

	return "", fmt.Errorf("channel not found in kyma-template spec")
}
