package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogCACertBundleDigest_FindsNonEmptyFile(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca-certificates.crt")
	content := []byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----\n")
	require.NoError(t, os.WriteFile(certPath, content, 0644))

	sum := sha256.Sum256(content)
	expectedDigest := fmt.Sprintf("%x", sum)

	var logged []map[string]any
	logger := slog.New(newCapturingHandler(&logged))

	overrideCertFiles := []string{certPath}
	logCACertBundleDigestFromPaths(logger, overrideCertFiles)

	require.Len(t, logged, 1)
	assert.Equal(t, slog.LevelInfo, logged[0]["level"])
	assert.Equal(t, certPath, logged[0]["path"])
	assert.Equal(t, expectedDigest, logged[0]["sha256"])
}

func TestLogCACertBundleDigest_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca-certificates.crt")
	require.NoError(t, os.WriteFile(certPath, []byte{}, 0644))

	var logged []map[string]any
	logger := slog.New(newCapturingHandler(&logged))

	logCACertBundleDigestFromPaths(logger, []string{certPath})

	require.Len(t, logged, 1)
	assert.Equal(t, slog.LevelWarn, logged[0]["level"])
}

func TestLogCACertBundleDigest_MissingFile(t *testing.T) {
	var logged []map[string]any
	logger := slog.New(newCapturingHandler(&logged))

	logCACertBundleDigestFromPaths(logger, []string{"/nonexistent/ca.crt"})

	require.Len(t, logged, 1)
	assert.Equal(t, slog.LevelWarn, logged[0]["level"])
}

func TestLogCACertBundleDigest_SkipsEmptyUsesNonEmpty(t *testing.T) {
	dir := t.TempDir()
	emptyPath := filepath.Join(dir, "empty.crt")
	goodPath := filepath.Join(dir, "good.crt")
	content := []byte("cert data")
	require.NoError(t, os.WriteFile(emptyPath, []byte{}, 0644))
	require.NoError(t, os.WriteFile(goodPath, content, 0644))

	sum := sha256.Sum256(content)
	expectedDigest := fmt.Sprintf("%x", sum)

	var logged []map[string]any
	logger := slog.New(newCapturingHandler(&logged))

	logCACertBundleDigestFromPaths(logger, []string{emptyPath, goodPath})

	require.Len(t, logged, 1)
	assert.Equal(t, slog.LevelInfo, logged[0]["level"])
	assert.Equal(t, goodPath, logged[0]["path"])
	assert.Equal(t, expectedDigest, logged[0]["sha256"])
}

// capturingHandler captures slog records for assertions.
type capturingHandler struct {
	records *[]map[string]any
}

func newCapturingHandler(out *[]map[string]any) *capturingHandler {
	return &capturingHandler{records: out}
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	entry := map[string]any{"level": r.Level, "msg": r.Message}
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.Any()
		return true
	})
	*h.records = append(*h.records, entry)
	return nil
}

func (h *capturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(name string) slog.Handler       { return h }
