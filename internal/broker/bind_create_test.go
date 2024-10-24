package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"code.cloudfoundry.org/lager"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type Kubeconfig struct {
	Users []User `yaml:"users"`
}

type User struct {
	Name string `yaml:"name"`
	User struct {
		Token string `yaml:"token"`
	} `yaml:"user"`
}

const (
	instanceID1      = "1"
	instanceID2      = "2"
	instanceID3      = "max-bindings"
	maxBindingsCount = 10
)

var httpServer *httptest.Server

type provider struct {
}

func (p *provider) K8sClientSetForRuntimeID(runtimeID string) (kubernetes.Interface, error) {
	return nil, fmt.Errorf("error")
}

func (p *provider) KubeconfigForRuntimeID(runtimeID string) ([]byte, error) {
	return []byte{}, nil
}

func TestCreateBindingEndpoint(t *testing.T) {
	t.Log("test create binding endpoint")

	// Given
	//// logger
	logs := logrus.New()
	logs.SetLevel(logrus.DebugLevel)
	logs.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	brokerLogger := lager.NewLogger("test")
	brokerLogger.RegisterSink(lager.NewWriterSink(logs.Writer(), lager.DEBUG))

	//// schema

	//// database
	db := storage.NewMemoryStorage()

	err := db.Instances().Insert(fixture.FixInstance(instanceID1))
	require.NoError(t, err)

	err = db.Instances().Insert(fixture.FixInstance(instanceID2))
	require.NoError(t, err)

	err = db.Instances().Insert(fixture.FixInstance(instanceID3))
	require.NoError(t, err)

	//// binding configuration
	bindingCfg := &BindingConfig{
		Enabled: true,
		BindablePlans: EnablePlans{
			fixture.PlanName,
		},
		MaxBindingsCount: maxBindingsCount,
	}

	//// api handler
	bindEndpoint := NewBind(*bindingCfg, db, logs, &provider{}, &provider{})

	// test relies on checking if got nil on kubeconfig provider but the instance got inserted either way
	t.Run("should INSERT binding despite error on k8s api call", func(t *testing.T) {
		// given
		_, err := db.Bindings().Get(instanceID1, "binding-id")
		require.Error(t, err)
		require.True(t, dberr.IsNotFound(err))

		// when
		_, err = bindEndpoint.Bind(context.Background(), instanceID1, "binding-id", domain.BindDetails{
			ServiceID: "123",
			PlanID:    fixture.PlanId,
		}, false)

		require.Error(t, err)

		binding, err := db.Bindings().Get(instanceID1, "binding-id")
		require.NoError(t, err)
		require.Equal(t, instanceID1, binding.InstanceID)
		require.Equal(t, "binding-id", binding.ID)

		require.NotNil(t, binding.ExpiresAt)
		require.Empty(t, binding.Kubeconfig)
	})
}

func TestCreatedBy(t *testing.T) {
	emptyStr := ""
	email := "john.smith@email.com"
	origin := "origin"
	tests := []struct {
		name     string
		context  BindingContext
		expected string
	}{
		{
			name:     "Both Email and Origin are nil",
			context:  BindingContext{Email: nil, Origin: nil},
			expected: "",
		},
		{
			name:     "Both Email and Origin are empty",
			context:  BindingContext{Email: &emptyStr, Origin: &emptyStr},
			expected: "",
		},
		{
			name:     "Origin is nil",
			context:  BindingContext{Email: &email, Origin: nil},
			expected: "john.smith@email.com",
		},
		{
			name:     "Origin is empty",
			context:  BindingContext{Email: &email, Origin: &emptyStr},
			expected: "john.smith@email.com",
		},
		{
			name:     "Email is nil",
			context:  BindingContext{Email: nil, Origin: &origin},
			expected: "origin",
		},
		{
			name:     "Email is empty",
			context:  BindingContext{Email: &emptyStr, Origin: &origin},
			expected: "origin",
		},
		{
			name:     "Both Email and Origin are set",
			context:  BindingContext{Email: &email, Origin: &origin},
			expected: "john.smith@email.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.context.CreatedBy())
		})
	}
}

func TestCreateSecondBindingWithTheSameIdButDifferentParams(t *testing.T) {
	// given
	instanceID := uuid.New().String()
	bindingID := uuid.New().String()
	instance := fixture.FixInstance(instanceID)
	bindingCfg := &BindingConfig{
		Enabled: true,
		BindablePlans: EnablePlans{
			instance.ServicePlanName,
		},
		ExpirationSeconds:    600,
		MaxExpirationSeconds: 7200,
		MinExpirationSeconds: 600,
		MaxBindingsCount:     10,
	}
	binding := fixture.FixBindingWithInstanceID(bindingID, instanceID)
	brokerStorage := storage.NewMemoryStorage()
	err := brokerStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	err = brokerStorage.Bindings().Insert(&binding)
	assert.NoError(t, err)

	svc := NewBind(*bindingCfg, brokerStorage, logrus.New(), nil, nil)
	params := BindingParams{
		ExpirationSeconds: 601,
	}
	rawParams, err := json.Marshal(params)
	assert.NoError(t, err)
	details := domain.BindDetails{
		RawParameters: rawParams,
	}

	// when
	_, err = svc.Bind(context.Background(), instanceID, bindingID, details, false)

	// then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binding already exists but with different parameters")
}

func TestCreateSecondBindingWithTheSameIdAndParams(t *testing.T) {
	// given
	const expiresAtLayout = "2006-01-02T15:04:05.0Z"
	instanceID := uuid.New().String()
	bindingID := uuid.New().String()
	instance := fixture.FixInstance(instanceID)
	bindingCfg := &BindingConfig{
		Enabled: true,
		BindablePlans: EnablePlans{
			instance.ServicePlanName,
		},
		ExpirationSeconds:    600,
		MaxExpirationSeconds: 7200,
		MinExpirationSeconds: 600,
		MaxBindingsCount:     10,
	}
	binding := fixture.FixBindingWithInstanceID(bindingID, instanceID)
	brokerStorage := storage.NewMemoryStorage()
	err := brokerStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	err = brokerStorage.Bindings().Insert(&binding)
	assert.NoError(t, err)

	svc := NewBind(*bindingCfg, brokerStorage, logrus.New(), nil, nil)
	params := BindingParams{
		ExpirationSeconds: 600,
	}
	rawParams, err := json.Marshal(params)
	assert.NoError(t, err)
	details := domain.BindDetails{
		RawParameters: rawParams,
	}

	// when
	resp, err := svc.Bind(context.Background(), instanceID, bindingID, details, false)

	// then
	assert.NoError(t, err)
	assert.Equal(t, binding.ExpiresAt.Format(expiresAtLayout), resp.Metadata.ExpiresAt)
}

func TestCreateSecondBindingWithTheSameIdAndParamsNotExplicitlyDefined(t *testing.T) {
	// given
	const expiresAtLayout = "2006-01-02T15:04:05.0Z"
	instanceID := uuid.New().String()
	bindingID := uuid.New().String()
	instance := fixture.FixInstance(instanceID)
	bindingCfg := &BindingConfig{
		Enabled: true,
		BindablePlans: EnablePlans{
			instance.ServicePlanName,
		},
		ExpirationSeconds:    600,
		MaxExpirationSeconds: 7200,
		MinExpirationSeconds: 600,
		MaxBindingsCount:     10,
	}
	binding := fixture.FixBindingWithInstanceID(bindingID, instanceID)
	brokerStorage := storage.NewMemoryStorage()
	err := brokerStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	err = brokerStorage.Bindings().Insert(&binding)
	assert.NoError(t, err)

	svc := NewBind(*bindingCfg, brokerStorage, logrus.New(), nil, nil)

	// when
	resp, err := svc.Bind(context.Background(), instanceID, bindingID, domain.BindDetails{}, false)

	// then
	assert.NoError(t, err)
	assert.Equal(t, binding.ExpiresAt.Format(expiresAtLayout), resp.Metadata.ExpiresAt)
}
