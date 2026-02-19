package main

import (
	"bytes"
	"log/slog"
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
		expectedLog    string
	}{
		"should expire instance": {
			modifyInstance: func(i *internal.Instance) {},
			config: Config{
				PlanID: broker.TrialPlanID,
			},
			expectedLog: "Instances: 1, to expire: 1, left non-expired: 0, suspension under way: 0, just marked expired: 1, failures: 0",
		},
		"should not expire already expired instance": {
			modifyInstance: func(i *internal.Instance) {
				i.ExpiredAt = ptr.Time(time.Now())
			},
			config: Config{
				PlanID: broker.TrialPlanID,
			},
			expectedLog: "Instances: 0, to expire: 0, left non-expired: 0, suspension under way: 0, just marked expired: 0, failures: 0",
		},
		"should not expire instance before expiration period": {
			modifyInstance: func(i *internal.Instance) {},
			config: Config{
				PlanID:           broker.TrialPlanID,
				ExpirationPeriod: 1 * time.Hour,
			},
			expectedLog: "Instances: 1, to expire: 0, left non-expired: 1, suspension under way: 0, just marked expired: 0, failures: 0",
		},
	} {
		t.Run(tn, func(t *testing.T) {
			cw := &captureWriter{buf: &bytes.Buffer{}}
			handler := slog.NewTextHandler(cw, nil)
			logger := slog.New(handler)
			slog.SetDefault(logger)

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

			err = svc.PerformCleanup()
			assert.NoError(t, err)

			assert.Contains(t, cw.buf.String(), tc.expectedLog)
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

type captureWriter struct {
	buf *bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}
