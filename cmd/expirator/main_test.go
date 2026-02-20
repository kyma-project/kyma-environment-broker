package main

import (
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
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

	config := expiratorTestConfig()
	cleanupContainer, err := docker.CreateDBContainer(internal.ContainerCreateRequest{
		Port:          config.Port,
		User:          config.User,
		Password:      config.Password,
		Name:          config.Name,
		Host:          config.Host,
		ContainerName: "test-expirator",
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
		modifyInstance func(*internal.Instance)
		config         Config
		expectedResult Result
	}{
		"should expire instance": {
			modifyInstance: func(i *internal.Instance) {},
			config: Config{
				PlanID: broker.TrialPlanID,
			},
			expectedResult: Result{
				count:                    1,
				instancesToExpireCount:   1,
				instancesToBeLeftCount:   0,
				suspensionsAcceptedCount: 0,
				onlyMarkedAsExpiredCount: 1,
				failuresCount:            0,
			},
		},
		"should not expire instance in dry run mode": {
			modifyInstance: func(i *internal.Instance) {},
			config: Config{
				PlanID: broker.TrialPlanID,
				DryRun: true,
			},
			expectedResult: Result{
				count:                    1,
				instancesToExpireCount:   1,
				instancesToBeLeftCount:   0,
				suspensionsAcceptedCount: 0,
				onlyMarkedAsExpiredCount: 0,
				failuresCount:            0,
			},
		},
		"should not expire already expired instance": {
			modifyInstance: func(i *internal.Instance) {
				i.ExpiredAt = ptr.Time(time.Now())
			},
			config: Config{
				PlanID: broker.TrialPlanID,
			},
			expectedResult: Result{
				count:                    0,
				instancesToExpireCount:   0,
				instancesToBeLeftCount:   0,
				suspensionsAcceptedCount: 0,
				onlyMarkedAsExpiredCount: 0,
				failuresCount:            0,
			},
		},
		"should not expire instance before expiration period": {
			modifyInstance: func(i *internal.Instance) {},
			config: Config{
				PlanID:           broker.TrialPlanID,
				ExpirationPeriod: 1 * time.Hour,
			},
			expectedResult: Result{
				count:                    1,
				instancesToExpireCount:   0,
				instancesToBeLeftCount:   1,
				suspensionsAcceptedCount: 0,
				onlyMarkedAsExpiredCount: 0,
				failuresCount:            0,
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			storageCleanup, db, err := storage.GetStorageForTests(expiratorTestConfig())
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, storageCleanup())
			}()

			instance := fixture.FixInstance("i1")
			instance.ServicePlanID = broker.TrialPlanID
			tc.modifyInstance(&instance)
			err = db.Instances().Insert(instance)
			require.NoError(t, err)

			op := fixture.FixProvisioningOperation("o1", "i1")
			err = db.Operations().InsertOperation(op)
			require.NoError(t, err)

			err = db.Instances().UpdateInstanceLastOperation("i1", "o1")
			require.NoError(t, err)

			svc := newCleanupService(
				tc.config,
				&mockBrokerClient{},
				db.Instances(),
			)

			result, err := svc.PerformCleanup()
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func expiratorTestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "FIPS-compl1antPwd!",
		Port:            "5431",
		Name:            "test-expirator",
		SSLMode:         "disable",
		SecretKey:       "################################",
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: 4 * time.Minute,
	}
}

type mockBrokerClient struct{}

func (m *mockBrokerClient) SendExpirationRequest(_ internal.Instance) (bool, error) {
	return false, nil
}
