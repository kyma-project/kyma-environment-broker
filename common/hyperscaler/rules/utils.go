package rules

import (
	"fmt"
	"os"
	"strings"
)

func RemoveWhitespaces(s string) string {
	var replacer = strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"\f", "")
	return replacer.Replace(s)
}

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
