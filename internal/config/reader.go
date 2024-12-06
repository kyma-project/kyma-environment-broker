package config

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace                 = "kcp-system"
	runtimeVersionLabelPrefix = "runtime-version-"
	kebConfigLabel            = "keb-config-runtime-configuration"
	defaultConfigKey          = "default"
)

type ConfigMapReader struct {
	ctx           context.Context
	k8sClient     client.Client
	logger        *slog.Logger
	configMapName string
}

func NewConfigMapReader(ctx context.Context, k8sClient client.Client, logger *slog.Logger, cmName string) ConfigReader {
	return &argoReader{target: &ConfigMapReader{
		ctx:           ctx,
		k8sClient:     k8sClient,
		logger:        logger,
		configMapName: cmName,
	}}
}

func (r *ConfigMapReader) Read(planName string) (string, error) {
	r.logger.Info(fmt.Sprintf("getting configuration for %v plan", planName))

	cfgMap, err := r.getConfigMap()
	if err != nil {
		return "", fmt.Errorf("while getting configuration configmap: %w", err)
	}
	cfgString, err := r.getConfigStringForPlanOrDefaults(cfgMap, planName)
	if err != nil {
		return "", fmt.Errorf("while getting configuration string: %w", err)
	}

	return cfgString, nil
}

func (r *ConfigMapReader) getConfigMap() (*coreV1.ConfigMap, error) {
	cfgMap := &coreV1.ConfigMap{}
	err := r.k8sClient.Get(r.ctx, client.ObjectKey{Namespace: namespace, Name: r.configMapName}, cfgMap)
	if errors.IsNotFound(err) {
		return nil, fmt.Errorf("configmap %s with configuration does not exist", r.configMapName)
	}
	return cfgMap, err
}

func (r *ConfigMapReader) getConfigStringForPlanOrDefaults(cfgMap *coreV1.ConfigMap, planName string) (string, error) {
	cfgString, exists := cfgMap.Data[planName]
	if !exists {
		r.logger.Info(fmt.Sprintf("configuration for plan %v does not exist. Using default values", planName))
		cfgString, exists = cfgMap.Data[defaultConfigKey]
		if !exists {
			return "", fmt.Errorf("default configuration does not exist")
		}
	}
	return cfgString, nil
}

type argoReader struct {
	target ConfigReader
}

func (r *argoReader) Read(planName string) (string, error) {
	content, err := r.target.Read(planName)
	if err != nil {
		return "", err
	}
	// a workaround for the issue with the Argo CD, see https://github.com/argoproj/argo-cd/pull/4729/files
	content = strings.Replace(content, "Kind:", "kind:", -1)
	return content, nil
}
