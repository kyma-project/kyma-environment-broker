package postsql_test

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type migrationOrder int

const (
	Up migrationOrder = iota
	Down
)

func brokerStorageTestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "test",
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
		ContainerName: "keb-storage-tests",
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

	exitVal = m.Run()
}

func GetStorageForTests() (func() error, storage.BrokerStorage, error) {
	config := brokerStorageTestConfig()
	storage, connection, err := storage.NewFromConfig(config, events.Config{}, storage.NewEncrypter(config.SecretKey), logrus.StandardLogger())
	if err != nil {
		return nil, nil, fmt.Errorf("while creating storage: %w", err)
	}
	if connection == nil {
		return nil, nil, fmt.Errorf("connection is nil")
	}
	if storage == nil {
		return nil, nil, fmt.Errorf("storage is nil")
	}

	failOnIncorrectDB(connection, config)
	failOnNotEmptyDb(connection)

	err = runMigrations(connection, Up)
	if err != nil {
		return nil, nil, fmt.Errorf("while applying migration files: %w", err)
	}
	log.Println("db created")

	cleanup := func() error {
		failOnIncorrectDB(connection, config)
		fmt.Println("cleaning up")
		err := runMigrations(connection, Down)
		if err != nil {
			return fmt.Errorf("failed to clear DB tables: %w", err)
		}
		fmt.Println("cleaned up")
		return nil
	}

	return cleanup, storage, nil
}

func runMigrations(connection *dbr.Connection, order migrationOrder) error {
	if order != Up && order != Down {
		return fmt.Errorf("unknown migration order")
	}

	migrations := "./../../../../resources/keb/migrations/"
	files, err := os.ReadDir(migrations)
	if err != nil {
		return fmt.Errorf("while reading migration data: %w in directory :%s", err, migrations)
	}

	suffix := ""
	if order == Down {
		suffix = "down.sql"
		slices.Reverse(files)
	}

	if order == Up {
		suffix = "up.sql"
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			content, err := os.ReadFile(migrations + file.Name())
			if err != nil {
				return fmt.Errorf("while reading migration files: %w file: %s", err, file.Name())
			}
			if _, err = connection.Exec(string(content)); err != nil {
				return fmt.Errorf("while applying migration files: %w file: %s", err, file.Name())
			}
		}
	}

	return nil
}

func failOnIncorrectDB(db *dbr.Connection, config storage.Config) {
	if db == nil {
		panic("db is nil")
	}
	row := db.QueryRow("SELECT CURRENT_DATABASE();")
	var result string
	err := row.Scan(&result)
	if err != nil {
		panic("cannot check if db has test prefix")
	}
	has := strings.HasPrefix(result, "test")
	if !has {
		panic("db has not test prefix")
	}
	equal := strings.EqualFold(result, config.Name)
	if !equal {
		panic(fmt.Sprintf("db: %s is not equal to config: %s", result, config.Name))
	}
}

func failOnNotEmptyDb(db *dbr.Connection) {
	tableExists := func(table string) (bool, error) {
		var rowResult string
		result := db.QueryRow(fmt.Sprintf("SELECT to_regclass('%s.%s')", "public", table))
		err := result.Scan(&rowResult)
		if err != nil {
			return false, err
		}
		return rowResult != "", nil
	}

	exists, err := tableExists(postsql.OperationTableName)
	if err != nil {
		panic(fmt.Sprintf("cannot verify if table %s exists: %s", postsql.OperationTableName, err.Error()))
	}
	if exists {
		panic(fmt.Sprintf("in db, data for %s are in table", postsql.OperationTableName))
	}

	exists, err = tableExists(postsql.InstancesTableName)
	if err != nil {
		panic(fmt.Sprintf("cannot verify if table %s exists: %s", postsql.InstancesTableName, err.Error()))
	}
	if exists {
		panic(fmt.Sprintf("in db, data for %s are in table", postsql.InstancesTableName))
	}
}
