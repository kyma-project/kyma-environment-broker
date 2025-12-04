package postsql

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"k8s.io/apimachinery/pkg/util/wait"
)

type EncryptionModeStats struct {
	postsql.Factory
}

func NewEncryptionModeStats(sess postsql.Factory) *EncryptionModeStats {
	return &EncryptionModeStats{
		Factory: sess,
	}
}

func (stats *EncryptionModeStats) GetEncryptionModeStatsForInstances() (map[string]int, error) {
	sess := stats.Factory.NewReadSession()
	var (
		rows    map[string]int
		lastErr error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		rows, lastErr = sess.GetEncryptionModeStatsForInstances()
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return rows, lastErr
	}

	return rows, nil
}

func (stats *EncryptionModeStats) GetEncryptionModeStatsForOperations() (map[string]int, error) {
	sess := stats.Factory.NewReadSession()
	var (
		rows    map[string]int
		lastErr error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		rows, lastErr = sess.GetEncryptionModeStatsForOperations()
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return rows, lastErr
	}

	return rows, nil
}

func (stats *EncryptionModeStats) GetEncryptionModeStatsForBindings() (map[string]int, error) {
	sess := stats.Factory.NewReadSession()
	var (
		rows    map[string]int
		lastErr error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		rows, lastErr = sess.GetEncryptionModeStatsForBindings()
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return rows, lastErr
	}

	return rows, nil
}
