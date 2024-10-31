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
)

const (
	expiresAtLayout = "2006-01-02T15:04:05.0Z"
)

type BindingConfig struct {
	Enabled              bool          `envconfig:"default=false"`
	BindablePlans        EnablePlans   `envconfig:"default=aws"`
	ExpirationSeconds    int           `envconfig:"default=600"`
	MaxExpirationSeconds int           `envconfig:"default=7200"`
	MinExpirationSeconds int           `envconfig:"default=600"`
	MaxBindingsCount     int           `envconfig:"default=10"`
	CreateBindingTimeout time.Duration `envconfig:"default=15s"`
}

type BindEndpoint struct {
	config           BindingConfig
	instancesStorage storage.Instances
	bindingsStorage  storage.Bindings

	serviceAccountBindingManager broker.BindingsManager

	log logrus.FieldLogger
}

type BindingContext struct {
	Email  *string `json:"email,omitempty"`
	Origin *string `json:"origin,omitempty"`
}

func (b *BindingContext) CreatedBy() string {
	if b.Email != nil && *b.Email != "" {
		return *b.Email
	} else if b.Origin != nil && *b.Origin != "" {
		return *b.Origin
	}
	return ""
}

type BindingParams struct {
	ExpirationSeconds int `json:"expiration_seconds,omit"`
}

type Credentials struct {
	Kubeconfig string `json:"kubeconfig"`
}

func NewBind(cfg BindingConfig, db storage.BrokerStorage, log logrus.FieldLogger, clientProvider broker.ClientProvider, kubeconfigProvider broker.KubeconfigProvider) *BindEndpoint {
	return &BindEndpoint{config: cfg,
		instancesStorage:             db.Instances(),
		bindingsStorage:              db.Bindings(),
		log:                          log.WithField("service", "BindEndpoint"),
		serviceAccountBindingManager: broker.NewServiceAccountBindingsManager(clientProvider, kubeconfigProvider),
	}
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

	var bindingContext BindingContext
	if len(details.RawContext) != 0 {
		err = json.Unmarshal(details.RawContext, &bindingContext)
		if err != nil {
			message := fmt.Sprintf("failed to unmarshal context: %s", err)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
		}
	}

	var parameters BindingParams
	if len(details.RawParameters) != 0 {
		err = json.Unmarshal(details.RawParameters, &parameters)
		if err != nil {
			message := fmt.Sprintf("failed to unmarshal parameters: %s", err)
			message = strings.Replace(message, "json: ", "", 1)
			message = strings.Replace(message, "Go struct field BindingParams.", "", 1)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
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

	bindingFromDB, err := b.bindingsStorage.Get(instanceID, bindingID)
	if err != nil && !dberr.IsNotFound(err) {
		message := fmt.Sprintf("failed to get Kyma binding from storage: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
	}
	if bindingFromDB != nil {
		if bindingFromDB.ExpirationSeconds != int64(expirationSeconds) {
			message := fmt.Sprintf("binding already exists but with different parameters")
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusConflict, message)
		}
		if bindingFromDB.ExpiresAt.After(time.Now()) {
			return domain.Binding{
				IsAsync:       false,
				AlreadyExists: true,
				Credentials: Credentials{
					Kubeconfig: bindingFromDB.Kubeconfig,
				},
				Metadata: domain.BindingMetadata{
					ExpiresAt: bindingFromDB.ExpiresAt.Format(expiresAtLayout),
				},
			}, nil
		}
	}

	bindingList, err := b.bindingsStorage.ListByInstanceID(instanceID)
	if err != nil {
		message := fmt.Sprintf("failed to list Kyma bindings: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
	}

	bindingCount := len(bindingList)
	message := fmt.Sprintf("reaching the maximum (%d) number of non expired bindings for instance %s", b.config.MaxBindingsCount, instanceID)
	if bindingCount == b.config.MaxBindingsCount-1 {
		b.log.Infof(message)
	}
	if bindingCount >= b.config.MaxBindingsCount {
		expiredCount := 0
		for _, binding := range bindingList {
			if binding.ExpiresAt.Before(time.Now()) {
				expiredCount++
			}
		}
		if (bindingCount - expiredCount) == (b.config.MaxBindingsCount - 1) {
			b.log.Infof(message)
		}
		if (bindingCount - expiredCount) >= b.config.MaxBindingsCount {
			message := fmt.Sprintf("maximum number of non expired bindings reached: %d", b.config.MaxBindingsCount)
			b.log.Infof(message+" for instance %s", instanceID)
			return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
		}
	}

	var kubeconfig string
	binding := &internal.Binding{
		ID:         bindingID,
		InstanceID: instanceID,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		ExpirationSeconds: int64(expirationSeconds),
		ExpiresAt:         time.Now().Add(time.Duration(expirationSeconds) * time.Second),
		CreatedBy:         bindingContext.CreatedBy(),
	}

	err = b.bindingsStorage.Insert(binding)
	switch {
	case dberr.IsAlreadyExists(err):
		message := fmt.Sprintf("failed to insert Kyma binding into storage: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
	case err != nil:
		message := fmt.Sprintf("failed to insert Kyma binding into storage: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
	}

	// create kubeconfig for the instance
	var expiresAt time.Time
	kubeconfig, expiresAt, err = b.serviceAccountBindingManager.Create(ctx, instance, bindingID, expirationSeconds)
	if err != nil {
		message := fmt.Sprintf("failed to create a Kyma binding using service account's kubeconfig: %s", err)
		b.log.Errorf("for instance %s %s", instanceID, message)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusBadRequest, message)
	}

	binding.ExpiresAt = expiresAt
	binding.Kubeconfig = kubeconfig

	err = b.bindingsStorage.Update(binding)
	if err != nil {
		message := fmt.Sprintf("failed to update Kyma binding in storage: %s", err)
		return domain.Binding{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
	}
	b.log.Infof("Successfully created binding %s for instance %s", bindingID, instanceID)

	return domain.Binding{
		IsAsync: false,
		Credentials: Credentials{
			Kubeconfig: kubeconfig,
		},
		Metadata: domain.BindingMetadata{
			ExpiresAt: binding.ExpiresAt.Format(expiresAtLayout),
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
