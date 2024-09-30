package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	broker "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BindingConfig struct {
	Enabled       bool        `envconfig:"default=false"`
	BindablePlans EnablePlans `envconfig:"default=aws"`
}

type BindEndpoint struct {
	config           BindingConfig
	instancesStorage storage.Instances

	tokenRequestBindingManager broker.BindingsManager
	gardenerBindingsManager    broker.BindingsManager

	log logrus.FieldLogger
}

type BindingParams struct {
	TokenRequest bool `json:"token_request,omit"`
}

type Credentials struct {
	Kubeconfig string `json:"kubeconfig"`
}

func NewBind(cfg BindingConfig, instanceStorage storage.Instances, log logrus.FieldLogger, clientProvider broker.ClientProvider, kubeconfigProvider broker.KubeconfigProvider, gardenerClient client.Client, tokenExpirationSeconds int) *BindEndpoint {
	return &BindEndpoint{config: cfg, instancesStorage: instanceStorage, log: log.WithField("service", "BindEndpoint"),
		tokenRequestBindingManager: broker.NewTokenRequestBindingsManager(clientProvider, kubeconfigProvider, tokenExpirationSeconds),
		gardenerBindingsManager:    broker.NewGardenerBindingManager(gardenerClient, tokenExpirationSeconds),
	}
}

type BindingData struct {
	Username string
	Password string
}

var dummyCredentials = BindingData{
	Username: "admin",
	Password: "admin1234",
}

// Bind creates a new service binding
//
//	PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *BindEndpoint) Bind(ctx context.Context, instanceID, bindingID string, details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {
	b.log.Infof("Bind instanceID: %s", instanceID)
	b.log.Infof("Bind parameters: %s", string(details.RawParameters))
	b.log.Infof("Bind context: %s", string(details.RawContext))
	b.log.Infof("Bind asyncAllowed: %v", asyncAllowed)

	if !b.config.Enabled {
		return domain.Binding{}, fmt.Errorf("not supported")
	}

	instance, err := b.instancesStorage.GetByID(instanceID)
	switch {
	case dberr.IsNotFound(err):
		return domain.Binding{}, apiresponses.ErrInstanceDoesNotExist
	case err != nil:
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get instance %s", instanceID), http.StatusInternalServerError, fmt.Sprintf("failed to get instance %s", instanceID))
	}

	if !b.IsPlanBindable(instance.ServicePlanName) {
		return domain.Binding{}, apiresponses.NewFailureResponseBuilder(
			errors.New("binding is not supported"), http.StatusUnprocessableEntity, "binding is not supported",
		).WithErrorKey("BindingNotSupported").Build()
	}

	var parameters BindingParams
	err = json.Unmarshal(details.RawParameters, &parameters)
	if err != nil {
		message := fmt.Sprintf("failed to unmarshal parameters: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
	}

	var kubeconfig string
	if parameters.TokenRequest {
		// get kubeconfig for the instance
		kubeconfig, err = b.tokenRequestBindingManager.Create(ctx, instance, bindingID)
		if err != nil {
			return domain.Binding{}, fmt.Errorf("failed to create kyma binding using token requests: %s", err)
		}
	} else {
		kubeconfig, err = b.gardenerBindingsManager.Create(ctx, instance, bindingID)
		if err != nil {
			return domain.Binding{}, fmt.Errorf("failed to create kyma binding using adminkubeconfig gardener subresource: %s", err)
		}
	}

	return domain.Binding{
		IsAsync: false,
		Credentials: Credentials{
			Kubeconfig: kubeconfig,
		},
	}, nil
}

func (b *BindEndpoint) IsPlanBindable(planName string) bool {
	planNameLowerCase := strings.ToLower(planName)
	for _, p := range b.config.BindablePlans {
		if strings.ToLower(p) == planNameLowerCase {
			return true
		}
	}
	return false
}
