package postsql

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"k8s.io/apimachinery/pkg/util/wait"
)

type TimeZones struct {
	postsql.Factory
}

func NewTimeZones(sess postsql.Factory) *TimeZones {
	return &TimeZones{
		Factory: sess,
	}
}

func (timeZones *TimeZones) GetTimeZone() (string, error) {
	sess := timeZones.Factory.NewReadSession()
	var (
		timeZone string
		lastErr  dberr.Error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		timeZone, lastErr = sess.GetTimeZone()
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", lastErr
	}

	return timeZone, nil
}
