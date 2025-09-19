package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
)

const (
	connRetries               = 30
	tempMigrationsPathPattern = "tmp-migrations-*"
	newMigrationsSrc          = "new-migrations"
	oldMigrationsSrc          = "migrations"
)

//go:generate mockery --name=FileSystem
type FileSystem interface {
	Open(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
	Create(name string) (*os.File, error)
	Chmod(name string, mode os.FileMode) error
	Copy(dst io.Writer, src io.Reader) (int64, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

//go:generate mockery --name=MyFileInfo
type MyFileInfo interface {
	Name() string       // base name of the file
	Size() int64        // length in bytes for regular files; system-dependent for others
	Mode() os.FileMode  // file mode bits
	ModTime() time.Time // modification time
	IsDir() bool        // abbreviation for Mode().IsDir()
	Sys() any           // underlying data source (can return nil)
}

type osFS struct{}

type migrationScript struct {
	fs FileSystem
}

func (osFS) Open(name string) (*os.File, error) {
	return os.Open(name)
}
func (osFS) Create(name string) (*os.File, error) {
	return os.Create(name)
}
func (osFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
func (osFS) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}
func (osFS) Copy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}
func (osFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	migrateErr := invokeMigration()
	if migrateErr != nil {
		slog.Info(fmt.Sprintf("while invoking migration: %s", migrateErr))
	}

	// continue with cleanup
	err := cleaner.Halt()

	if err != nil || migrateErr != nil {
		slog.Error(fmt.Sprintf("error during migration: %s", migrateErr))
		slog.Error(fmt.Sprintf("error during cleanup: %s", err))
		os.Exit(-1)
	}
}

func invokeMigration() error {
	envs := []string{
		"DB_USER", "DB_HOST", "DB_NAME", "DB_PORT",
		"DB_PASSWORD", "DIRECTION",
	}

	for _, env := range envs {
		_, present := os.LookupEnv(env)
		if !present {
			return fmt.Errorf("ERROR: %s is not set", env)
		}
	}

	direction := os.Getenv("DIRECTION")
	switch direction {
	case "up":
		slog.Info("# MIGRATION UP #")
	case "down":
		slog.Info("# MIGRATION DOWN #")
	default:
		return errors.New("ERROR: DIRECTION variable accepts only two values: up or down")
	}

	dbName := os.Getenv("DB_NAME")

	_, present := os.LookupEnv("DB_SSL")
	if present {
		sslMode := os.Getenv("DB_SSL")
		dbName = fmt.Sprintf("%s?sslmode=%s", dbName, sslMode)
		if sslMode != "disable" {
			_, present := os.LookupEnv("DB_SSLROOTCERT")
			if present {
				dbName = fmt.Sprintf("%s&sslrootcert=%s", dbName, os.Getenv("DB_SSLROOTCERT"))
			}
		}
	}

	hostPort := net.JoinHostPort(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"))

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		hostPort,
		dbName,
	)

	slog.Info("# WAITING FOR CONNECTION WITH DATABASE #")
	db, err := sql.Open("pgx", connectionString)

	for i := 0; i < connRetries && err != nil; i++ {
		slog.Error(fmt.Sprintf("Error while connecting to the database, %s. Retrying step", err))
		db, err = sql.Open("pgx", connectionString)
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT ESTABLISH CONNECTION TO DATABASE WITH CONNECTION STRING: %w", err)
	}
	slog.Info("# CONNECTION WITH DATABASE ESTABLISHED #")
	slog.Info("# STARTING TO COPY MIGRATION FILES #")

	migrationExecPath, err := os.MkdirTemp("/migrate", tempMigrationsPathPattern)
	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE TEMPORARY DIRECTORY FOR MIGRATION: %w", err)
	}
	defer os.RemoveAll(migrationExecPath)

	ms := migrationScript{
		fs: osFS{},
	}
	slog.Info("# LOADING MIGRATION FILES FROM CONFIGMAP #")
	err = ms.copyDir(newMigrationsSrc, migrationExecPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("# NO MIGRATION FILES PROVIDED BY THE CONFIGMAP, SKIPPING STEP #")
		} else {
			return fmt.Errorf("# COULD NOT COPY MIGRATION FILES PROVIDED BY THE CONFIGMAP: %w", err)
		}
	} else {
		slog.Info("# LOADING MIGRATION FILES FROM CONFIGMAP DONE #")
	}
	slog.Info("# LOADING EMBEDDED MIGRATION FILES FROM THE SCHEMA-MIGRATOR IMAGE #")
	err = ms.copyDir(oldMigrationsSrc, migrationExecPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("# NO MIGRATION FILES EMBEDDED TO THE SCHEMA-MIGRATOR IMAGE, SKIPPING STEP #")
		} else {
			return fmt.Errorf("# COULD NOT COPY EMBEDDED MIGRATION FILES FROM THE SCHEMA-MIGRATOR IMAGE: %w", err)
		}
	} else {
		slog.Info("# LOADING EMBEDDED MIGRATION FILES FROM THE SCHEMA-MIGRATOR IMAGE DONE #")
	}

	slog.Info("# INITIALIZING DRIVER #")
	
	// Try standard migration with panic recovery for FIPS issues
	var driver database.Driver
	var driverErr error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicMsg := fmt.Sprintf("%v", r)
				if strings.Contains(panicMsg, "FIPS") || strings.Contains(panicMsg, "hmac") {
					slog.Info(fmt.Sprintf("# FIPS compliance panic detected: %s #", panicMsg))
					slog.Info("# Using direct SQL migration approach to avoid FIPS issues #")
					driverErr = fmt.Errorf("FIPS_FALLBACK_REQUIRED")
				} else {
					// Re-panic if it's not a FIPS issue
					panic(r)
				}
			}
		}()
		driver, driverErr = createMigrationDriver(db)
	}()
	
	// If we need FIPS fallback, use direct SQL migration
	if driverErr != nil && driverErr.Error() == "FIPS_FALLBACK_REQUIRED" {
		return performDirectSQLMigration(db, migrationExecPath, direction)
	}

	for i := 0; i < connRetries && driverErr != nil; i++ {
		// Check if error is FIPS-related
		if strings.Contains(driverErr.Error(), "FIPS") || strings.Contains(driverErr.Error(), "hmac") {
			slog.Info("# FIPS compliance issue detected, using direct SQL migration approach #")
			return performDirectSQLMigration(db, migrationExecPath, direction)
		}
		
		slog.Error(fmt.Sprintf("Error during driver initialization, %s. Retrying step", driverErr))
		
		// Try again with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicMsg := fmt.Sprintf("%v", r)
					if strings.Contains(panicMsg, "FIPS") || strings.Contains(panicMsg, "hmac") {
						driverErr = fmt.Errorf("FIPS_FALLBACK_REQUIRED")
					} else {
						panic(r)
					}
				}
			}()
			driver, driverErr = createMigrationDriver(db)
		}()
		
		if driverErr != nil && driverErr.Error() == "FIPS_FALLBACK_REQUIRED" {
			return performDirectSQLMigration(db, migrationExecPath, direction)
		}
		
		time.Sleep(100 * time.Millisecond)
	}

	if driverErr != nil {
		return fmt.Errorf("# COULD NOT CREATE DATABASE CONNECTION: %w", driverErr)
	}
	
	slog.Info("# DRIVER INITIALIZED #")
	slog.Info("# STARTING MIGRATION #")

	migrationPath := fmt.Sprintf("file:///%s", migrationExecPath)

	migrateInstance, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("error during migration initialization: %w", err)
	}

	defer func(migrateInstance *migrate.Migrate) {
		err, _ := migrateInstance.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("error during migrate instance close: %s", err))
		}
	}(migrateInstance)
	migrateInstance.Log = &Logger{}

	if direction == "up" {
		err = migrateInstance.Up()
	} else if direction == "down" {
		err = migrateInstance.Down()
	}

	if err != nil && !errors.Is(migrate.ErrNoChange, err) {
		return fmt.Errorf("during migration: %w", err)
	} else if errors.Is(migrate.ErrNoChange, err) {
		slog.Info("# NO CHANGES DETECTED #")
	}

	slog.Info("# MIGRATION DONE #")

	currentMigrationVer, _, err := migrateInstance.Version()
	if err == migrate.ErrNilVersion {
		slog.Info("# NO ACTIVE MIGRATION VERSION #")
	} else if err != nil {
		return fmt.Errorf("during acquiring active migration version: %w", err)
	}

	slog.Info(fmt.Sprintf("# CURRENT ACTIVE MIGRATION VERSION: %d #", currentMigrationVer))
	return nil
}

type Logger struct{}

func (l *Logger) Printf(format string, v ...interface{}) {
	fmt.Printf("Executed "+format, v...)
}

func (l *Logger) Verbose() bool {
	return false
}

func (m *migrationScript) copyFile(src, dst string) error {
	rd, err := m.fs.Open(src)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer rd.Close()

	wr, err := m.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer wr.Close()

	_, err = m.fs.Copy(wr, rd)
	if err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	srcInfo, err := m.fs.Stat(src)
	if err != nil {
		return fmt.Errorf("retrieving fileinfo: %w", err)
	}

	return m.fs.Chmod(dst, srcInfo.Mode())
}

func (m *migrationScript) copyDir(src, dst string) error {
	files, err := m.fs.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		srcFile := path.Join(src, file.Name())
		dstFile := path.Join(dst, file.Name())
		if fileExists(dstFile) {
			slog.Info(fmt.Sprintf("file %s already exists, skipping", dstFile))
			continue
		}
		fileExt := filepath.Ext(srcFile)
		if fileExt == ".sql" {
			err = m.copyFile(srcFile, dstFile)
			if err != nil {
				return fmt.Errorf("error during: %w", err)
			}
		}
	}

	return nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// createMigrationDriver attempts to create a postgres migration driver
func createMigrationDriver(db *sql.DB) (database.Driver, error) {
	return postgres.WithInstance(db, &postgres.Config{})
}

// performDirectSQLMigration executes migrations directly using SQL without golang-migrate
// This is a FIPS-compliant fallback that avoids HMAC issues in the migration library
func performDirectSQLMigration(db *sql.DB, migrationPath, direction string) error {
	slog.Info("# STARTING DIRECT SQL MIGRATION (FIPS-COMPLIANT) #")
	
	// Create migrations table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint NOT NULL,
			dirty boolean NOT NULL,
			PRIMARY KEY (version)
		);`
	
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	// Get current migration version
	var currentVersion int64
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&currentVersion, &dirty)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}
	
	if dirty {
		return fmt.Errorf("database is in dirty state, manual intervention required")
	}
	
	// Read migration files
	files, err := os.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}
	
	// Execute migrations based on direction
	if direction == "up" {
		return executeUpMigrations(db, migrationPath, files, currentVersion)
	} else {
		return executeDownMigrations(db, migrationPath, files, currentVersion)
	}
}

// executeUpMigrations runs up migrations
func executeUpMigrations(db *sql.DB, migrationPath string, files []os.DirEntry, currentVersion int64) error {
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}
		
		// Extract version from filename (format: YYYYMMDDHHMMSS_name.up.sql)
		versionStr := strings.Split(file.Name(), "_")[0]
		var version int64
		_, err := fmt.Sscanf(versionStr, "%d", &version)
		if err != nil {
			slog.Info(fmt.Sprintf("Skipping file with invalid version format: %s", file.Name()))
			continue
		}
		
		// Skip if already applied
		if version <= currentVersion {
			continue
		}
		
		slog.Info(fmt.Sprintf("Applying migration: %s", file.Name()))
		
		// Read SQL file
		sqlPath := filepath.Join(migrationPath, file.Name())
		sqlBytes, err := os.ReadFile(sqlPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}
		
		// Execute migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		
		// Mark as dirty
		_, err = tx.Exec("INSERT INTO schema_migrations (version, dirty) VALUES ($1, true) ON CONFLICT (version) DO UPDATE SET dirty = true", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to mark migration as dirty: %w", err)
		}
		
		// Execute migration SQL
		_, err = tx.Exec(string(sqlBytes))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
		}
		
		// Mark as clean
		_, err = tx.Exec("UPDATE schema_migrations SET dirty = false WHERE version = $1", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to mark migration as clean: %w", err)
		}
		
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit migration transaction: %w", err)
		}
		
		slog.Info(fmt.Sprintf("Successfully applied migration: %s", file.Name()))
	}
	
	slog.Info("# DIRECT SQL MIGRATION UP COMPLETED #")
	return nil
}

// executeDownMigrations runs down migrations
func executeDownMigrations(db *sql.DB, migrationPath string, files []os.DirEntry, currentVersion int64) error {
	// Find the down migration for the current version
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".down.sql") {
			continue
		}
		
		// Extract version from filename
		versionStr := strings.Split(file.Name(), "_")[0]
		var version int64
		_, err := fmt.Sscanf(versionStr, "%d", &version)
		if err != nil {
			continue
		}
		
		// Only process the current version's down migration
		if version != currentVersion {
			continue
		}
		
		slog.Info(fmt.Sprintf("Applying down migration: %s", file.Name()))
		
		// Read SQL file
		sqlPath := filepath.Join(migrationPath, file.Name())
		sqlBytes, err := os.ReadFile(sqlPath)
		if err != nil {
			return fmt.Errorf("failed to read down migration file %s: %w", file.Name(), err)
		}
		
		// Execute migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		
		// Mark as dirty
		_, err = tx.Exec("UPDATE schema_migrations SET dirty = true WHERE version = $1", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to mark migration as dirty: %w", err)
		}
		
		// Execute down migration SQL
		_, err = tx.Exec(string(sqlBytes))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute down migration %s: %w", file.Name(), err)
		}
		
		// Remove migration record
		_, err = tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to remove migration record: %w", err)
		}
		
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit down migration transaction: %w", err)
		}
		
		slog.Info(fmt.Sprintf("Successfully applied down migration: %s", file.Name()))
		break
	}
	
	slog.Info("# DIRECT SQL MIGRATION DOWN COMPLETED #")
	return nil
}
