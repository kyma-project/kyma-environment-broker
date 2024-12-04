package postsql

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/gocraft/dbr"

	"github.com/lib/pq"
)

const (
	schemaName                 = "public"
	InstancesTableName         = "instances"
	OperationTableName         = "operations"
	OrchestrationTableName     = "orchestrations"
	RuntimeStateTableName      = "runtime_states"
	SubaccountStatesTableName  = "subaccount_states"
	CreatedAtField             = "created_at"
	InstancesArchivedTableName = "instances_archived"
	BindingsTableName          = "bindings"
)

// InitializeDatabase opens database connection and initializes schema if it does not exist
func InitializeDatabase(connectionURL string, retries int) (*dbr.Connection, error) {
	connection, err := WaitForDatabaseAccess(connectionURL, retries, 300*time.Millisecond)
	if err != nil {
		return nil, err
	}

	initialized, err := CheckIfDatabaseInitialized(connection)
	if err != nil {
		closeDBConnection(connection)
		return nil, fmt.Errorf("failed to check if database is initialized: %w", err)
	}
	if initialized {
		slog.Info("Database already initialized")
		return connection, nil
	}

	return connection, nil
}

func closeDBConnection(db *dbr.Connection) {
	err := db.Close()
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to close database connection: %s", err.Error()))
	}
}

const TableNotExistsError = "42P01"

func CheckIfDatabaseInitialized(db *dbr.Connection) (bool, error) {
	checkQuery := fmt.Sprintf(`SELECT '%s.%s'::regclass;`, schemaName, InstancesTableName)

	row := db.QueryRow(checkQuery)

	var tableName string
	err := row.Scan(&tableName)

	if err != nil {
		psqlErr, converted := err.(*pq.Error)

		if converted && psqlErr.Code == TableNotExistsError {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if database is initialized: %w", err)
	}

	return tableName == InstancesTableName, nil
}

func WaitForDatabaseAccess(connString string, retryCount int, sleepTime time.Duration) (*dbr.Connection, error) {
	var connection *dbr.Connection
	var err error

	re := regexp.MustCompile(`password=.*?\s`)
	slog.Info(re.ReplaceAllString(connString, ""))

	for ; retryCount > 0; retryCount-- {
		connection, err = dbr.Open("postgres", connString, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid connection string: %w", err)
		}

		err = connection.Ping()
		if err == nil {
			return connection, nil
		}
		slog.Warn(fmt.Sprintf("Database Connection failed: %s", err.Error()))

		err = connection.Close()
		if err != nil {
			slog.Info("Failed to close database ...")
		}

		slog.Info(fmt.Sprintf("Failed to access database, waiting %v to retry...", sleepTime))
		time.Sleep(sleepTime)
	}

	return nil, fmt.Errorf("timeout waiting for database access")
}
