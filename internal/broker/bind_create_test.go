package broker

import (
	"testing"

	"context"
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/stretchr/testify/assert"

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
	instanceID1          = "1"
	instanceID2          = "2"
	instanceID3          = "max-bindings"
)

var httpServer *httptest.Server

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
	}

	//// api handler
	bindEndpoint := NewBind(*bindingCfg, db.Instances(), db.Bindings(), logs, nil, nil) // test relies on checking if got nil on kubeconfig provider but the instance got inserted either way

	t.Run("should INSERT binding despite error on k8s api call", func(t *testing.T) {
		defer func() {
			r := recover()

			// then
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			binding, err := db.Bindings().Get(instanceID1, "binding-id")
			require.NoError(t, err)
			require.Equal(t, instanceID1, binding.InstanceID)
			require.Equal(t, "binding-id", binding.ID)
		}()

		// given
		_, err := db.Bindings().Get(instanceID1, "binding-id")
		require.Error(t, err)
		require.True(t, dberr.IsNotFound(err))

		// when
		_, _ = bindEndpoint.Bind(context.Background(), instanceID1, "binding-id", domain.BindDetails{
			ServiceID: "123",
			PlanID:    fixture.PlanId,
		}, false)
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
