package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
	_ "github.com/lib/pq"
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
		slog.Error(fmt.Sprintf("while invoking migration: %s", migrateErr))
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

	connectionString, err := buildConnectionString()
	if err != nil {
		return err
	}

	slog.Info("# WAITING FOR CONNECTION WITH DATABASE #")
	db, err := sql.Open("postgres", connectionString)

	for i := 0; i < connRetries && err != nil; i++ {
		slog.Error(fmt.Sprintf("Error while connecting to the database, %s. Retrying step", err))
		db, err = sql.Open("postgres", connectionString)
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
	defer func() { _ = os.RemoveAll(migrationExecPath) }()

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
	driver, err := postgres.WithInstance(db, &postgres.Config{})

	for i := 0; i < connRetries && err != nil; i++ {
		slog.Error(fmt.Sprintf("Error during driver initialization, %s. Retrying step", err))
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("# COULD NOT CREATE DATABASE CONNECTION: %w", err)
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

func buildConnectionString() (string, error) {

	dbName := os.Getenv("DB_NAME")

	_, present := os.LookupEnv("DB_SSL")
	if present {
		dbName = fmt.Sprintf("%s?sslmode=%s", dbName, os.Getenv("DB_SSL"))

		_, present := os.LookupEnv("DB_SSLROOTCERT")
		if present {
			dbName = fmt.Sprintf("%s&sslrootcert=%s", dbName, os.Getenv("DB_SSLROOTCERT"))
		}
	}

	// hard-wire time zone as UTC to avoid issues with different time zones
	dbName += "&timezone=UTC"

	hostPort := net.JoinHostPort(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"))

	// URL-encode username and password to handle special characters
	encodedUser := url.QueryEscape(os.Getenv("DB_USER"))
	encodedPassword := url.QueryEscape(os.Getenv("DB_PASSWORD"))

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s/%s",
		encodedUser,
		encodedPassword,
		hostPort,
		dbName,
	)
	return connectionString, nil
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
	defer func() { _ = rd.Close() }()

	wr, err := m.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = wr.Close() }()

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
