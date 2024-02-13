package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func brokerStorageTestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "test",
		Port:            "5431",
		Name:            "test-e2e",
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

	if os.Getenv("DB_IN_MEMORY") != "true" {
		config := brokerStorageTestConfig()

		docker, err := internal.NewDockerHandler()
		if err != nil {
			log.Fatal(err)
		}
		defer func(docker *internal.DockerHelper) {
			err := docker.CloseDockerClient()
			if err != nil {
				log.Fatal(err)
			}
		}(docker)

		cleanupContainer, err := docker.CreateDBContainer(internal.ContainerCreateRequest{
			Port:          config.Port,
			User:          config.User,
			Password:      config.Password,
			Name:          config.Name,
			Host:          config.Host,
			ContainerName: "e2e-tests",
			Image:         "postgres:11",
		})
		defer func() {
			if cleanupContainer != nil {
				err := cleanupContainer()
				if err != nil {
					log.Fatal(err)
				}
			}
		}()
		if err != nil {
			log.Fatal(err)
		}
	}

	exitVal = m.Run()
}

func GetStorageForE2ETests() (func() error, storage.BrokerStorage, error) {
	if os.Getenv("DB_IN_MEMORY") == "true" {
		return nil, storage.NewMemoryStorage(), nil
	}
	return storage.GetStorageForTest(brokerStorageTestConfig())
}
