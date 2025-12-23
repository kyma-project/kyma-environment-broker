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

type Secrets interface {
	Create(secret v1.Secret) error
	Delete(secret v1.Secret) error
}

type SecretClient struct {
	client client.Client
	log    *slog.Logger
}

func NewSecretClient(client client.Client, log *slog.Logger) *SecretClient {
	return &SecretClient{client: client, log: log}
}

func (c *SecretClient) Create(secret v1.Secret) error {
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.client.Create(context.Background(), &secret)
		if err != nil {
			if apiErrors.IsAlreadyExists(err) {
				err = c.Delete(secret)
				if err != nil {
					return false, fmt.Errorf("while deleting a secret: %w", err)
				}
				err = c.client.Create(context.Background(), &secret)
				if err != nil {
					return false, fmt.Errorf("while creating secret: %w", err)
				}
				return true, nil
			}
			c.log.Error(fmt.Sprintf("while creating secret: %v", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for secret creation: %w", err)
	}
	return nil
}

func (c *SecretClient) Delete(secret v1.Secret) error {
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.client.Delete(context.Background(), &secret)
		if err != nil {
			if apiErrors.IsNotFound(err) {
				c.log.Warn("secret not found")
				return true, nil
			}
			c.log.Error(fmt.Sprintf("while creating secret: %v", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for secret delete: %w", err)
	}
	return nil
}
