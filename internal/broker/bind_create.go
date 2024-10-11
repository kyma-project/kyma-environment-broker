package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	broker "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BindingConfig struct {
	Enabled              bool        `envconfig:"default=false"`
	BindablePlans        EnablePlans `envconfig:"default=aws"`
	ExpirationSeconds    int         `envconfig:"default=600"`
	MaxExpirationSeconds int         `envconfig:"default=7200"`
	MinExpirationSeconds int         `envconfig:"default=600"`
}

type BindEndpoint struct {
	config           BindingConfig
	instancesStorage storage.Instances
	bindingsStorage  storage.Bindings

	serviceAccountBindingManager broker.BindingsManager
	gardenerBindingsManager      broker.BindingsManager

	log logrus.FieldLogger
}

type BindingParams struct {
	ServiceAccount    bool `json:"service_account,omit"`
	ExpirationSeconds int  `json:"expiration_seconds,omit"`
}

type Credentials struct {
	Kubeconfig string `json:"kubeconfig"`
}

func NewBind(cfg BindingConfig, instanceStorage storage.Instances, bindingsStorage storage.Bindings, log logrus.FieldLogger, clientProvider broker.ClientProvider, kubeconfigProvider broker.KubeconfigProvider, gardenerClient client.Client) *BindEndpoint {
	return &BindEndpoint{config: cfg,
		instancesStorage:             instanceStorage,
		bindingsStorage:              bindingsStorage,
		log:                          log.WithField("service", "BindEndpoint"),
		serviceAccountBindingManager: broker.NewServiceAccountBindingsManager(clientProvider, kubeconfigProvider),
		gardenerBindingsManager:      broker.NewGardenerBindingManager(gardenerClient),
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
	if len(details.RawParameters) != 0 {
		err = json.Unmarshal(details.RawParameters, &parameters)
		if err != nil {
			message := fmt.Sprintf("failed to unmarshal parameters: %s", err)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
		}
	}

	expirationSeconds := b.config.ExpirationSeconds
	if parameters.ExpirationSeconds != 0 {
		if parameters.ExpirationSeconds > b.config.MaxExpirationSeconds {
			message := fmt.Sprintf("expiration_seconds cannot be greater than %d", b.config.MaxExpirationSeconds)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
		}
		if parameters.ExpirationSeconds < b.config.MinExpirationSeconds {
			message := fmt.Sprintf("expiration_seconds cannot be less than %d", b.config.MinExpirationSeconds)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
		}
		expirationSeconds = parameters.ExpirationSeconds
	}

	var kubeconfig string
	var expiresAt time.Time
	binding := &internal.Binding{
		ID:         bindingID,
		InstanceID: instanceID,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		ExpirationSeconds: int64(expirationSeconds),
	}
	if parameters.ServiceAccount {
		// get kubeconfig for the instance
		kubeconfig, expiresAt, err = b.serviceAccountBindingManager.Create(ctx, instance, bindingID, expirationSeconds)
		if err != nil {
			message := fmt.Sprintf("failed to create a Kyma binding using service account's kubeconfig: %s", err)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
		}
		binding.BindingType = internal.BINDING_TYPE_SERVICE_ACCOUNT
	} else {
		kubeconfig, expiresAt, err = b.gardenerBindingsManager.Create(ctx, instance, bindingID, expirationSeconds)
		if err != nil {
			message := fmt.Sprintf("failed to create a Kyma binding using adminkubeconfig gardener subresource: %s", err)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
		}
		binding.BindingType = internal.BINDING_TYPE_ADMIN_KUBECONFIG
	}

	binding.ExpiresAt = expiresAt
	binding.Kubeconfig = kubeconfig

	err = b.bindingsStorage.Insert(binding)
	if err != nil {
		message := fmt.Sprintf("failed to insert Kyma binding into storage: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)

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
