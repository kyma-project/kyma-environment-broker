package storage

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"golang.org/x/exp/slices"
)

type migrationOrder int

const (
	Up migrationOrder = iota
	Down
)

func GetStorageForTest(config Config) (func() error, BrokerStorage, error) {
	connectionURL := config.ConnectionURL()
	return GetStorageForTestUsingConnectionURL(config, connectionURL)
}

func GetStorageForTestUsingConnectionURL(config Config, connectionURL string) (func() error, BrokerStorage, error) {
	storageForTests, connection, err := NewFromConfigAndConnectionURL(config, events.Config{}, NewEncrypter(config.SecretKey), connectionURL)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating storage: %w", err)
	}
	if connection == nil {
		return nil, nil, fmt.Errorf("connection is nil")
	}
	if storageForTests == nil {
		return nil, nil, fmt.Errorf("storage is nil")
	}

	failOnIncorrectDB(connection, config)
	failOnNotEmptyDb(connection)

	err = runMigrations(connection, Up)
	if err != nil {
		return nil, nil, fmt.Errorf("while applying migration files: %w", err)
	}

	cleanup := func() (error error) {
		defer func() {
			err = connection.Close()
			if err != nil {
				error = fmt.Errorf("failed to close connection: %w", err)
			}
		}()
		failOnIncorrectDB(connection, config)
		err = runMigrations(connection, Down)
		if err != nil {
			return fmt.Errorf("failed to clear DB tables: %w", err)
		}
		return
	}

	return cleanup, storageForTests, nil
}

func runMigrations(connection *dbr.Connection, order migrationOrder) error {
	_, currentPath, _, _ := runtime.Caller(0)
	migrationsPath := fmt.Sprintf("%s/resources/keb/migrations/", path.Join(path.Dir(currentPath), "../../"))

	if order != Up && order != Down {
		return fmt.Errorf("unknown migration order")
	}

	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("while reading migration data: %w in directory :%s", err, migrationsPath)
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
			content, err := os.ReadFile(migrationsPath + file.Name())
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

func failOnIncorrectDB(db *dbr.Connection, config Config) {
	if db == nil {
		panic("db is nil")
	}
	row := db.QueryRow("SELECT CURRENT_DATABASE();")
	var result string
	err := row.Scan(&result)
	if err != nil || result == "" {
		panic(fmt.Sprintf("cannot check if db has test prefix. %s", err.Error()))
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
		var exists bool
		result := db.QueryRow(fmt.Sprintf(
			`
			SELECT EXISTS (
			    SELECT FROM
			        pg_tables
			    WHERE
			        schemaname = 'public' AND
			        tablename  = '%s'
			);`,
			table))
		err := result.Scan(&exists)
		if err != nil {
			return false, err
		}
		return exists, nil
	}

	for _, table := range []string{postsql.OperationTableName, postsql.SubaccountStatesTableName, postsql.InstancesTableName} {
		exists, err := tableExists(table)
		if err != nil {
			panic(fmt.Sprintf("cannot verify if table %s exists: %s", table, err.Error()))
		}
		if exists {
			panic(fmt.Sprintf("in db, data for %s are in table", table))
		}
	}
}
