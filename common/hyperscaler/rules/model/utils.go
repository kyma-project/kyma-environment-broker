package model

import (
	"fmt"
	"os"
)

func CreateTempFile(content string) (string, error) {
	tmpfile, err := os.CreateTemp("", "test*.yaml")

	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %s", err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %s", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %s", err)
	}

	return tmpfile.Name(), nil
}
