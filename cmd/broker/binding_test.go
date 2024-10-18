package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

func TestBinding(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()
	bid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "eu-central-1"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
		`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)

	b, _ := io.ReadAll(resp.Body)
	suite.Log(string(b))
	suite.Log(resp.Status)

	respRuntimes := suite.CallAPI("GET", "/runtimes?bindings=true", "")
	b, _ = io.ReadAll(respRuntimes.Body)
	suite.Log(string(b))
	suite.Log(resp.Status)

	t.Run("should return 400 when expiration seconds parameter is string instead of int", func(t *testing.T) {
		resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": "600"
				}
               }`)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
