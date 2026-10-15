package postsql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gocraft/dbr"
	"github.com/gocraft/dbr/dialect"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	schemaName                 = "public"
	InstancesTableName         = "instances"
	OperationTableName         = "operations"
	SubaccountStatesTableName  = "subaccount_states"
	CreatedAtField             = "created_at"
	InstancesArchivedTableName = "instances_archived"
	BindingsTableName          = "bindings"
	ActionsTableName           = "actions"
)

// InitializeDatabase opens database connection and initializes schema if it does not exist
func InitializeDatabase(connectionURL string, retries int) (*dbr.Connection, error) {
	connection, err := WaitForDatabaseAccess(connectionURL, retries, time.Second)
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
		var psqlErr *pgconn.PgError
		if errors.As(err, &psqlErr) && psqlErr.Code == TableNotExistsError {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if database is initialized: %w", err)
	}

	return tableName == InstancesTableName, nil
}

func WaitForDatabaseAccess(connString string, retryCount int, sleepTime time.Duration) (*dbr.Connection, error) {
	slog.Info("Connecting to database")

	connConfig, err := pgx.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	afterConnect := stdlib.OptionAfterConnect(func(_ context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "timestamptz",
			OID:   pgtype.TimestamptzOID,
			Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
		})
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "timestamp",
			OID:   pgtype.TimestampOID,
			Codec: &pgtype.TimestampCodec{ScanLocation: time.UTC},
		})
		return nil
	})

	for ; retryCount > 0; retryCount-- {
		sqlDB := stdlib.OpenDB(*connConfig, afterConnect)
		connection := &dbr.Connection{DB: sqlDB, Dialect: dialect.PostgreSQL, EventReceiver: &dbr.NullEventReceiver{}}

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
