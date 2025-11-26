package config

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	error2 "github.com/kyma-project/kyma-environment-broker/internal/error"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace        = "kcp-system"
	defaultConfigKey = "default"
)

type ConfigMapReader struct {
	ctx       context.Context
	k8sClient client.Client
	logger    *slog.Logger
}

func NewConfigMapReader(ctx context.Context, k8sClient client.Client, logger *slog.Logger) Reader {
	return &ConfigMapReader{
		ctx:       ctx,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

func (r *ConfigMapReader) Read(objectName, configKey string) (string, error) {
	r.logger.Info(fmt.Sprintf("getting %s configuration from %s configmap", configKey, objectName))

	cfgMap, err := r.getConfigMap(objectName)
	if err != nil {
		return "", error2.NewTemporaryError("while getting configmap: %s", err.Error())
	}
	cfgString, err := r.getConfigStringOrDefaults(cfgMap.Data, configKey)
	if err != nil {
		return "", error2.NewTemporaryError("while getting configuration string: %s", err.Error())
	}

	// a workaround for the issue with the ArgoCD, see https://github.com/argoproj/argo-cd/pull/4729/files
	cfgString = strings.Replace(cfgString, "Kind:", "kind:", -1)

	return cfgString, nil
}

func (r *ConfigMapReader) getConfigMap(configMapName string) (*coreV1.ConfigMap, error) {
	cfgMap := &coreV1.ConfigMap{}
	err := r.k8sClient.Get(r.ctx, client.ObjectKey{Namespace: namespace, Name: configMapName}, cfgMap)
	if errors.IsNotFound(err) {
		return nil, fmt.Errorf("configmap %s does not exist in %s namespace", configMapName, namespace)
	}
	return cfgMap, err
}

func (r *ConfigMapReader) getConfigStringOrDefaults(configMapData map[string]string, configKey string) (string, error) {
	cfgString, exists := configMapData[configKey]
	if !exists {
		r.logger.Info(fmt.Sprintf("configuration key %s does not exist. Using default values", configKey))
		cfgString, exists = configMapData[defaultConfigKey]
		if !exists {
			return "", fmt.Errorf("both configuration key '%s' and default configuration does not exist. The ConfigMap is not properly configured.", configKey)
		}
	}
	return cfgString, nil
}
