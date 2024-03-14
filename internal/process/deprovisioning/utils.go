package deprovisioning

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/sirupsen/logrus"
)

const (
	testInstanceID      = "58f8c703-1756-48ab-9299-a847974d1fee"
	testOperationID     = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	testSubAccountID    = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
	testGlobalAccountID = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
)

func handleError(stepName string, operation internal.Operation, err error,
	log logrus.FieldLogger, msg string) (internal.Operation, time.Duration, error) {

	if kebError.IsTemporaryError(err) {
		if time.Since(operation.CreatedAt) < 30*time.Minute {
			log.Errorf("%s: %s. Retry...", msg, err)
			return operation, 10 * time.Second, nil
		}
	}

	log.Errorf("Step %s failed: %s.", stepName, err)
	return operation, 0, nil
}
