package main

import (
	"fmt"
	"io/fs"
	"os"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/mocks"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func Test_migrationScript_copyFile(t *testing.T) {
	t.Run("Should return error while opening source file fails", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Open", "src").Return(nil, fmt.Errorf("failed to open file"))
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyFile("src", "dst")

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while creating destination file fails", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(nil, fmt.Errorf("failed to create file"))
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyFile("src", "dst")

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while copying file fails", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(0), fmt.Errorf("failed to copy file"))
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyFile("src", "dst")

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while returning FileInfo fails", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfi := &mocks.MyFileInfo{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(65), nil)
		mfs.On("Stat", "src").Return(mfi, fmt.Errorf("failed to get FileInfo"))
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyFile("src", "dst")

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while changing the mode of the file fails", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfi := &mocks.MyFileInfo{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(65), nil)
		mfs.On("Stat", "src").Return(mfi, nil)
		mfi.On("Mode").Return(fs.FileMode(0666))
		mfs.On("Chmod", "dst", fs.FileMode(0666)).Return(fmt.Errorf("failed to change file mode"))
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyFile("src", "dst")

		// then
		assert.Error(t, err)
	})
	t.Run("Should not return error", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfi := &mocks.MyFileInfo{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(65), nil)
		mfs.On("Stat", "src").Return(mfi, nil)
		mfi.On("Mode").Return(fs.FileMode(0666))
		mfs.On("Chmod", "dst", fs.FileMode(0666)).Return(nil)
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyFile("src", "dst")

		// then
		assert.Nil(t, err)
	})
}

func Test_migrationScript_copyDir(t *testing.T) {
	t.Run("Should not return error", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("ReadDir", "src").Return([]fs.DirEntry{}, nil)
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyDir("src", "dst")

		// then
		assert.Nil(t, err)

	})
	t.Run("Should return error while reading directory fails", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("ReadDir", "src").Return(nil, fmt.Errorf("failed to read directory"))
		ms := &migrationScript{
			fs: mfs,
		}

		// when
		err := ms.copyDir("src", "dst")

		// then
		assert.Error(t, err)

	})
}

func Test_buildConnectionString(t *testing.T) {
	t.Run("Should handle special characters in password", func(t *testing.T) {
		// given
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "admin12345#@!:pass")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_NAME", "testdb")
		defer func() {
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_SSL")
		}()

		// when
		connString, err := buildConnectionString()

		// then
		assert.Nil(t, err)
		assert.Contains(t, connString, "postgres://testuser:admin12345%23%40%21%3Apass@localhost:5432/testdb")
		assert.Contains(t, connString, "timezone=UTC")
	})

	t.Run("Should build connection string without SSL", func(t *testing.T) {
		// given
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_HOST", "host")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_NAME", "dbname")
		defer func() {
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")
		}()

		// when
		connString, err := buildConnectionString()

		// then
		assert.Nil(t, err)
		assert.Equal(t, "postgres://user:pass@host:5432/dbname&timezone=UTC", connString)
	})

	t.Run("Should build connection string with SSL", func(t *testing.T) {
		// given
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_HOST", "host")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_NAME", "dbname")
		os.Setenv("DB_SSL", "require")
		defer func() {
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_SSL")
		}()

		// when
		connString, err := buildConnectionString()

		// then
		assert.Nil(t, err)
		assert.Equal(t, "postgres://user:pass@host:5432/dbname?sslmode=require&timezone=UTC", connString)
	})

	t.Run("Should build connection string with SSL and root certificate", func(t *testing.T) {
		// given
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_HOST", "host")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_NAME", "dbname")
		os.Setenv("DB_SSL", "require")
		os.Setenv("DB_SSLROOTCERT", "/path/to/cert")
		defer func() {
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_SSL")
			os.Unsetenv("DB_SSLROOTCERT")
		}()

		// when
		connString, err := buildConnectionString()

		// then
		assert.Nil(t, err)
		assert.Equal(t, "postgres://user:pass@host:5432/dbname?sslmode=require&sslrootcert=/path/to/cert&timezone=UTC", connString)
	})
}
