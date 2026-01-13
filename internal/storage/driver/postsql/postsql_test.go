package postsql_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func brokerStorageDatabaseTestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "FIPS-compl1antPwd!",
		Port:            "5430",
		Name:            "testbroker",
		SSLMode:         "disable",
		SecretKey:       "################################",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
}

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() {
		os.Exit(exitVal)
	}()

	config := brokerStorageDatabaseTestConfig()

	docker, err := internal.NewDockerHandler()
	fatalOnError(err)
	defer func(docker *internal.DockerHelper) {
		err := docker.CloseDockerClient()
		fatalOnError(err)
	}(docker)

	cleanupContainer, err := docker.CreateDBContainer(internal.ContainerCreateRequest{
		Port:          config.Port,
		User:          config.User,
		Password:      config.Password,
		Name:          config.Name,
		Host:          config.Host,
		ContainerName: "keb-storage-tests",
		Image:         internal.PostgresImage,
	})
	defer func() {
		if cleanupContainer != nil {
			err := cleanupContainer()
			fatalOnError(err)
		}
	}()
	fatalOnError(err)

	exitVal = m.Run()
}

func GetStorageForDatabaseTests() (func() error, storage.BrokerStorage, error) {
	return storage.GetStorageForTest(brokerStorageDatabaseTestConfig())
}

func GetStorageForDatabaseTestsWithEncrypter(encrypter *storage.Encrypter) (func() error, storage.BrokerStorage, error) {
	return storage.GetStorageForTestWithMutableEncrypter(brokerStorageDatabaseTestConfig(), encrypter)
}

func fatalOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
