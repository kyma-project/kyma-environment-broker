package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() {
		os.Exit(exitVal)
	}()

	docker, err := internal.NewDockerHandler()
	fatalOnError(err)
	defer func() {
		fatalOnError(docker.CloseDockerClient())
	}()

	config := deprovisionRetriggerTestConfig()
	cleanupContainer, err := docker.CreateDBContainer(internal.ContainerCreateRequest{
		Port:          config.Port,
		User:          config.User,
		Password:      config.Password,
		Name:          config.Name,
		Host:          config.Host,
		ContainerName: "test-deprovision-retrigger",
		Image:         internal.PostgresImage,
	})
	fatalOnError(err)
	defer func() {
		fatalOnError(cleanupContainer())
	}()

	exitVal = m.Run()
}

func TestCleanup(t *testing.T) {
	for tn, tc := range map[string]struct {
		dryRun         bool
		statusCode     int
		deprovisionErr error
		expectedResult Result
	}{
		"should retrigger deprovisioning for deleted instance": {
			dryRun:         false,
			statusCode:     http.StatusNotFound,
			deprovisionErr: nil,
			expectedResult: Result{
				instancesToDeprovisionAgain: 1,
				deprovisioningAccepted:      1,
				sanityFailedCount:           0,
				failuresCount:               0,
			},
		},
		"should not retrigger deprovisioning for deleted instance in dry run mode": {
			dryRun:         true,
			statusCode:     http.StatusNotFound,
			deprovisionErr: nil,
			expectedResult: Result{
				instancesToDeprovisionAgain: 1,
				deprovisioningAccepted:      0,
				sanityFailedCount:           0,
				failuresCount:               0,
			},
		},
		"should not retrigger deprovisioning when sanity returns 200": {
			dryRun:         false,
			statusCode:     http.StatusOK,
			deprovisionErr: nil,
			expectedResult: Result{
				instancesToDeprovisionAgain: 1,
				deprovisioningAccepted:      0,
				sanityFailedCount:           1,
				failuresCount:               0,
			},
		},
		"should count failure when deprovisioning returns error": {
			dryRun:         false,
			statusCode:     http.StatusNotFound,
			deprovisionErr: fmt.Errorf("deprovision failed"),
			expectedResult: Result{
				instancesToDeprovisionAgain: 1,
				deprovisioningAccepted:      0,
				sanityFailedCount:           0,
				failuresCount:               1,
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			storageCleanup, db, err := storage.GetStorageForTests(deprovisionRetriggerTestConfig())
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, storageCleanup())
			}()

			instance := fixture.FixInstance("i1")
			instance.DeletedAt = time.Now()
			err = db.Instances().Insert(instance)
			require.NoError(t, err)

			op := fixture.FixProvisioningOperation("o1", "i1")
			err = db.Operations().InsertOperation(op)
			require.NoError(t, err)

			err = db.Instances().UpdateInstanceLastOperation("i1", "o1")
			require.NoError(t, err)

			svc := newDeprovisionRetriggerService(
				Config{DryRun: tc.dryRun},
				&mockBrokerClient{statusCode: tc.statusCode, deprovisionErr: tc.deprovisionErr},
				db.Instances(),
			)

			result, err := svc.PerformCleanup()
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func deprovisionRetriggerTestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "FIPS-compl1antPwd!",
		Port:            "5434",
		Name:            "test-deprovision-retrigger",
		SSLMode:         "disable",
		SecretKey:       "################################",
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: 4 * time.Minute,
	}
}

type mockBrokerClient struct {
	statusCode     int
	deprovisionErr error
}

func (m *mockBrokerClient) Deprovision(instance internal.Instance) (string, error) {
	if m.deprovisionErr != nil {
		return "", m.deprovisionErr
	}
	return "op-id", nil
}

func (m *mockBrokerClient) GetInstanceRequest(instanceID string) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}
