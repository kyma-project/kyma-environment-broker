package broker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

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
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	assert.NoError(t, err)
	err = st.Bindings().Insert(&binding)
	assert.NoError(t, err)

	svc := NewBind(*bindingCfg, st.Instances(), st.Bindings(), logrus.New(), nil, nil)
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
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	assert.NoError(t, err)
	err = st.Bindings().Insert(&binding)
	assert.NoError(t, err)

	svc := NewBind(*bindingCfg, st.Instances(), st.Bindings(), logrus.New(), nil, nil)
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
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	assert.NoError(t, err)
	err = st.Bindings().Insert(&binding)
	assert.NoError(t, err)

	svc := NewBind(*bindingCfg, st.Instances(), st.Bindings(), logrus.New(), nil, nil)

	// when
	resp, err := svc.Bind(context.Background(), instanceID, bindingID, domain.BindDetails{}, false)

	// then
	assert.NoError(t, err)
	assert.Equal(t, binding.ExpiresAt.Format(expiresAtLayout), resp.Metadata.ExpiresAt)
}
