package hyperscalers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const testNamespace = "test-ns"

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// fixBinding creates a CredentialsBinding object with the given name, pointing to
// a secret with the same name in testNamespace.
func fixBinding(name string) *unstructured.Unstructured {
	cb := &gardener.CredentialsBinding{}
	cb.SetName(name)
	cb.SetNamespace(testNamespace)
	cb.SetGroupVersionKind(gardener.CredentialsBindingGVK)
	cb.SetLabels(map[string]string{"hyperscalerType": "aws"})
	cb.SetSecretRefName(name)
	cb.SetSecretRefNamespace(testNamespace)
	return &cb.Unstructured
}

// fixSecret creates a minimal secret object.
func fixSecret(name string) *unstructured.Unstructured {
	s := &unstructured.Unstructured{}
	s.SetName(name)
	s.SetNamespace(testNamespace)
	s.SetGroupVersionKind(gardener.SecretGVK)
	return s
}

func newTestGardenerClient(t *testing.T, objects ...*unstructured.Unstructured) *gardener.Client {
	t.Helper()
	dynClient := gardener.NewDynamicFakeClient()
	gc := gardener.NewClient(dynClient, testNamespace)

	ctx := context.Background()
	for _, obj := range objects {
		gvr := gardener.CredentialsBindingResource
		if obj.GroupVersionKind() == gardener.SecretGVK {
			gvr = gardener.SecretResource
		}
		_, err := dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
		require.NoError(t, err, "setup: creating object %s", obj.GetName())
	}
	return gc
}

const testLabelSelector = "hyperscalerType=aws"

// --- candidateIndices ---

func TestCandidateIndices_One(t *testing.T) {
	got := candidateIndices(1)
	assert.Equal(t, []int{0}, got)
}

func TestCandidateIndices_Two(t *testing.T) {
	got := candidateIndices(2)
	assert.Equal(t, []int{0, 1}, got)
}

func TestCandidateIndices_Ten(t *testing.T) {
	got := candidateIndices(10)
	// Must include 0 (first), 5 (middle), 9 (last)
	assert.Contains(t, got, 0)
	assert.Contains(t, got, 5)
	assert.Contains(t, got, 9)
	// At most 4 unique elements
	assert.LessOrEqual(t, len(got), 4)
	// No duplicates
	seen := make(map[int]struct{})
	for _, idx := range got {
		_, dup := seen[idx]
		assert.False(t, dup, "duplicate index %d", idx)
		seen[idx] = struct{}{}
	}
}

func TestCandidateIndices_Zero(t *testing.T) {
	got := candidateIndices(0)
	assert.Nil(t, got)
}

// --- withBindingRetry ---

func TestWithRetry_NoBindings(t *testing.T) {
	gc := newTestGardenerClient(t)
	_, err := WithRetry(context.Background(), gc, testLabelSelector, testLogger(),
		func(_ context.Context, _ *gardener.CredentialsBinding, _ *unstructured.Unstructured) (string, error) {
			return "ok", nil
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no credentials bindings found")
}

func TestWithRetry_FirstBindingSucceeds(t *testing.T) {
	gc := newTestGardenerClient(t,
		fixBinding("cb-0"),
		fixSecret("cb-0"),
	)
	result, err := WithRetry(context.Background(), gc, testLabelSelector, testLogger(),
		func(_ context.Context, cb *gardener.CredentialsBinding, _ *unstructured.Unstructured) (string, error) {
			return cb.GetName(), nil
		})
	require.NoError(t, err)
	assert.Equal(t, "cb-0", result)
}

func TestWithRetry_FirstSecretMissing_SecondSucceeds(t *testing.T) {
	// cb-0's secret does not exist; cb-1 has its secret
	gc := newTestGardenerClient(t,
		fixBinding("cb-0"),
		fixBinding("cb-1"),
		fixSecret("cb-1"),
	)
	result, err := WithRetry(context.Background(), gc, testLabelSelector, testLogger(),
		func(_ context.Context, cb *gardener.CredentialsBinding, _ *unstructured.Unstructured) (string, error) {
			return cb.GetName(), nil
		})
	require.NoError(t, err)
	assert.Equal(t, "cb-1", result)
}

func TestWithRetry_AllBindingsFail_AttemptFn(t *testing.T) {
	gc := newTestGardenerClient(t,
		fixBinding("cb-0"),
		fixSecret("cb-0"),
	)
	callErr := errors.New("invalid credentials")
	_, err := WithRetry(context.Background(), gc, testLabelSelector, testLogger(),
		func(_ context.Context, _ *gardener.CredentialsBinding, _ *unstructured.Unstructured) (string, error) {
			return "", callErr
		})
	require.Error(t, err)
	assert.ErrorIs(t, err, callErr)
}

func TestWithRetry_FirstAttemptFails_SecondSucceeds(t *testing.T) {
	// Two bindings, both secrets exist. Attempt fn fails for cb-0 but succeeds for any other.
	gc := newTestGardenerClient(t,
		fixBinding("cb-0"),
		fixBinding("cb-1"),
		fixSecret("cb-0"),
		fixSecret("cb-1"),
	)
	result, err := WithRetry(context.Background(), gc, testLabelSelector, testLogger(),
		func(_ context.Context, cb *gardener.CredentialsBinding, _ *unstructured.Unstructured) (string, error) {
			if cb.GetName() == "cb-0" {
				return "", fmt.Errorf("expired credentials for %s", cb.GetName())
			}
			return cb.GetName(), nil
		})
	require.NoError(t, err)
	assert.NotEqual(t, "cb-0", result)
}

func TestWithRetry_AllAttemptsExhausted(t *testing.T) {
	// Provide enough bindings so at least 2 unique candidates are tried, all fail.
	gc := newTestGardenerClient(t,
		fixBinding("cb-0"),
		fixBinding("cb-1"),
		fixBinding("cb-2"),
		fixSecret("cb-0"),
		fixSecret("cb-1"),
		fixSecret("cb-2"),
	)
	_, err := WithRetry(context.Background(), gc, testLabelSelector, testLogger(),
		func(_ context.Context, _ *gardener.CredentialsBinding, _ *unstructured.Unstructured) (string, error) {
			return "", errors.New("probe failed")
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all")
	assert.Contains(t, err.Error(), "attempts failed")
}
