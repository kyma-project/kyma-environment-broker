package hyperscalers

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// candidateIndices returns up to 4 deduplicated indices to try from a list of n items:
// first (0), middle (n/2), last (n-1), and one random index.
func candidateIndices(n int) []int {
	if n <= 0 {
		return nil
	}
	if n <= 4 {
		result := make([]int, n)
		for i := range result {
			result[i] = i
		}
		return result
	}
	seen := make(map[int]struct{}, 4)
	var result []int
	for _, idx := range []int{0, n / 2, n - 1, rand.IntN(n)} {
		if _, already := seen[idx]; already {
			continue
		}
		seen[idx] = struct{}{}
		result = append(result, idx)
	}
	return result
}

// WithRetry lists credentials bindings matching labelSelector, then tries each
// candidate index (first, middle, last, random — up to 4 unique attempts).
// For each candidate it fetches the Gardener secret and calls attempt. Returns the
// first successful result. If all attempts fail, returns the last error.
func WithRetry[T any](
	ctx context.Context,
	gc *gardener.Client,
	labelSelector string,
	log *slog.Logger,
	attempt func(ctx context.Context, binding *gardener.CredentialsBinding, secret *unstructured.Unstructured) (T, error),
) (T, error) {
	var zero T

	credentialsBindings, err := gc.GetCredentialsBindings(labelSelector)
	if err != nil {
		return zero, fmt.Errorf("while getting credentials bindings with selector %q: %w", labelSelector, err)
	}
	if credentialsBindings == nil || len(credentialsBindings.Items) == 0 {
		return zero, fmt.Errorf("no credentials bindings found for selector %q", labelSelector)
	}

	items := credentialsBindings.Items
	candidates := candidateIndices(len(items))
	var lastErr error

	for _, idx := range candidates {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		cb := gardener.NewCredentialsBinding(items[idx])
		log.Info("trying credentials binding", "name", cb.GetName(), "index", idx)

		secret, err := gc.GetSecret(cb.GetSecretRefNamespace(), cb.GetSecretRefName())
		if err != nil {
			log.Warn("failed to get secret for credentials binding", "name", cb.GetName(), "error", err)
			lastErr = err
			continue
		}

		result, err := attempt(ctx, cb, secret)
		if err != nil {
			log.Warn("credentials binding attempt failed", "name", cb.GetName(), "error", err)
			lastErr = err
			continue
		}

		return result, nil
	}

	return zero, fmt.Errorf("all %d credentials binding attempts failed for selector %q: %w", len(candidates), labelSelector, lastErr)
}
