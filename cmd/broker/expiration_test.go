package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

const expirationRequestPathFormat = "expire/service_instance/%s"

const trialProvisioningRequestBody = `{
"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
  "context": {
    "sm_operator_credentials": {
      "clientid": "sm-operator-client-id",
      "clientsecret": "sm-operator-client-secret",
      "url": "sm-operator-url",
      "sm_url": "sm-operator-url"
    },
    "globalaccount_id": "global-account-id",
    "subaccount_id": "subaccount-id",
    "user_id": "john.smith@email.com"
  },
  "parameters": {
    "name": "trial-test",
    "oidc": {
      "clientID": "client-id",
      "signingAlgs": ["PS512"],
      "issuerURL": "https://issuer.url.com"
    }
  }
}`

func TestExpiration(t *testing.T) {
	// before all
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()

	t.Run("should expire a trial instance", func(t *testing.T) {
		// given
		instanceID := uuid.NewString()
		resp := suite.CallAPI(http.MethodPut,
			fmt.Sprintf(provisioningRequestPathFormat, instanceID),
			trialProvisioningRequestBody)
		provisioningOpID := suite.DecodeOperationID(resp)
		suite.processProvisioningAndReconcilingByOperationID(provisioningOpID)

		// when
		resp = suite.CallAPI(http.MethodPut,
			fmt.Sprintf(expirationRequestPathFormat, instanceID),
			"")

		// then
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		suspensionOpID := suite.DecodeOperationID(resp)
		assert.NotEmpty(t, suspensionOpID)

		suite.WaitForOperationState(suspensionOpID, domain.InProgress)
		suite.MarkClusterConfigurationDeleted(instanceID)
		suite.FinishDeprovisioningOperationByProvisionerForGivenOpId(suspensionOpID)
		suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

		actualInstance := suite.GetInstance(instanceID)
		assert.True(t, actualInstance.IsExpired())
	})
}
