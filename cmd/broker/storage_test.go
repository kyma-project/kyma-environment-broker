package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func brokerStorageE2ETestConfig() storage.Config {
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

	if !dbInMemoryForE2ETests() {
		config := brokerStorageE2ETestConfig()

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
			ContainerName: "keb-e2e-tests",
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
