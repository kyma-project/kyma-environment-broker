package v1_client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout = time.Second * 10
)

type ConfigMaps interface {
	Get(name, namespace string) (*v1.ConfigMap, error)
	Create(configMap v1.ConfigMap) error
	Update(configMap v1.ConfigMap) error
	Delete(configMap v1.ConfigMap) error
}

type ConfigMapClient struct {
	client client.Client
	log    *slog.Logger
}

func NewConfigMapClient(client client.Client, log *slog.Logger) *ConfigMapClient {
	return &ConfigMapClient{client: client, log: log}
}

func (c *ConfigMapClient) Get(name, namespace string) (*v1.ConfigMap, error) {
	configMap := v1.ConfigMap{}
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.client.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, &configMap)
		if err != nil {
			if apiErrors.IsNotFound(err) {
				return false, err
			}
			c.log.Error(fmt.Sprintf("while creating config map: %v", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("while getting config map: %w", err)
	}
	return &configMap, nil
}

func (c *ConfigMapClient) Create(configMap v1.ConfigMap) error {
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.client.Create(context.Background(), &configMap)
		if err != nil {
			if apiErrors.IsAlreadyExists(err) {
				err = c.Update(configMap)
				if err != nil {
					return false, fmt.Errorf("while updating a config map: %w", err)
				}
				return true, nil
			}
			c.log.Error(fmt.Sprintf("while creating config map: %v", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while creating config map: %w", err)
	}
	return nil
}

func (c *ConfigMapClient) Update(configMap v1.ConfigMap) error {
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.client.Update(context.Background(), &configMap)
		if err != nil {
			c.log.Error(fmt.Sprintf("while updating config map: %v", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for config map update: %w", err)
	}
	return nil
}

func (c *ConfigMapClient) Delete(configMap v1.ConfigMap) error {
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.client.Delete(context.Background(), &configMap)
		if err != nil {
			if apiErrors.IsNotFound(err) {
				c.log.Warn("config map not found")
				return true, nil
			}
			c.log.Error(fmt.Sprintf("while deleting config map: %v", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for config map delete: %w", err)
	}
	return nil
}
