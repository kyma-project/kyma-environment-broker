package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ec2"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type GetAWSZonesStep struct {
	operationManager *process.OperationManager
	opStorage        storage.Operations
	gardenerClient   *gardener.Client
}

func NewGetAWSZonesStep(os storage.Operations, gardenerClient *gardener.Client) *GetAWSZonesStep {
	step := &GetAWSZonesStep{
		opStorage:      os,
		gardenerClient: gardenerClient,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *GetAWSZonesStep) Name() string {
	return "Get_AWS_Zones"
}

func (s *GetAWSZonesStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.ProvisioningParameters.PlatformProvider != pkg.AWS {
		log.Info("PlatformProvider is not AWS, skipping")
		return operation, 0, nil
	}
	if len(operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools) == 0 {
		log.Info("No additional worker node pools, skipping")
		return operation, 0, nil
	}

	if operation.ProvisioningParameters.Parameters.TargetSecret == nil {
		return s.operationManager.OperationFailed(operation, "target secret is missing", nil, log)
	}
	if operation.ProvisioningParameters.Parameters.Region == nil {
		return s.operationManager.OperationFailed(operation, "region is missing", nil, log)
	}

	secret, err := s.gardenerClient.GetSecret(*operation.ProvisioningParameters.Parameters.TargetSecret)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "unable to get secret", err, 10*time.Second, time.Minute, log)
	}
	accessKeyID, secretAccessKey, err := s.extractAWSCredentials(secret)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "failed to extract AWS credentials", err, log)
	}

	client, err := ec2.NewClient(context.Background(), accessKeyID, secretAccessKey, *operation.ProvisioningParameters.Parameters.Region)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "unable to create EC2 client", err, 10*time.Second, time.Minute, log)
	}
	for _, pool := range operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools {
		zones, err := client.AvailableZones(context.Background(), pool.MachineType)
		if err != nil {
			return s.operationManager.RetryOperation(operation, "unable to get available zones", err, 10*time.Second, time.Minute, log)
		}
		log.Info(fmt.Sprintf("Available zones for %s: %v", pool.MachineType, zones))
	}

	return operation, 0, nil
}

func (s *GetAWSZonesStep) extractAWSCredentials(secret *unstructured.Unstructured) (string, string, error) {
	data, found, err := unstructured.NestedStringMap(secret.Object, "data")
	if err != nil || !found {
		return "", "", fmt.Errorf("unable to extract data from secret: %w", err)
	}

	accessKeyID, ok := data["accessKeyID"]
	if !ok {
		return "", "", fmt.Errorf("secret does not contain accessKeyID")
	}
	secretAccessKey, ok := data["secretAccessKey"]
	if !ok {
		return "", "", fmt.Errorf("secret does not contain secretAccessKey")
	}

	return accessKeyID, secretAccessKey, nil
}
