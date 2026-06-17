package hyperscalers

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Client interface {
	AvailableZones(ctx context.Context, machineType string) ([]string, error)
	AvailableZonesCount(ctx context.Context, machineType string) (int, error)
}

type ClientFactory interface {
	NewFromSecret(ctx context.Context, secret *unstructured.Unstructured, region string) (Client, error)
}
