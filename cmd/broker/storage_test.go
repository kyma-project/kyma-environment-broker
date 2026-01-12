package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

var cleanupContainer func() error

func brokerStorageE2ETestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "FIPS-compl1antPwd!",
		Port:            "5431",
		Name:            "test-e2e",
		SSLMode:         "disable",
		SecretKey:       "################################",
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: 4 * time.Minute,
	}
}

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() {
		os.Exit(exitVal)
	}()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if !dbInMemoryForE2ETests() {
		config := brokerStorageE2ETestConfig()

		docker, err := internal.NewDockerHandler()
		fatalOnError(err, log)
		defer func(docker *internal.DockerHelper) {
			err := docker.CloseDockerClient()
			fatalOnError(err, log)
		}(docker)

		cleanupContainer, err = docker.CreateDBContainer(internal.ContainerCreateRequest{
			Port:          config.Port,
			User:          config.User,
			Password:      config.Password,
			Name:          config.Name,
			Host:          config.Host,
			ContainerName: "keb-e2e-tests",
			Image:         internal.PostgresImage,
		})
		defer func() {
			if cleanupContainer != nil {
				err := cleanupContainer()
				fatalOnError(err, log)
			}
		}()
		fatalOnError(err, log)
	}

	exitVal = m.Run()
}

func GetStorageForE2ETests() (func() error, storage.BrokerStorage, error) {
	if dbInMemoryForE2ETests() {
		return nil, storage.NewMemoryStorage(), nil
	}
	return storage.GetStorageForTest(brokerStorageE2ETestConfig())
}

func dbInMemoryForE2ETests() bool {
	v, _ := strconv.ParseBool(os.Getenv("DB_IN_MEMORY_FOR_E2E_TESTS"))
	if v {
		fmt.Println("running e2e test on database in memory")
	} else {
		fmt.Println("running e2e test on real postgres database")
	}
	return v
}
