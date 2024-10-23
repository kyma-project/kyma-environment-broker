package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

type ErrorResponse struct {
	Description string `json:"description"`
}

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

		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		var apiError ErrorResponse
		err = json.Unmarshal(b, &apiError)
		assert.NoError(t, err)

		assert.Equal(t, "failed to unmarshal parameters: cannot unmarshal string into expiration_seconds of type int", apiError.Description)
	})

	t.Run("should return 200 when creating a second binding with the same id and params as an existing one", func(t *testing.T) {
		bid = uuid.New().String()
		resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }`)
		resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }`)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// "expiration_seconds": 600 is a default value from the config in tests
		resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should return 409 when creating a second binding with the same id as an existing one but different params", func(t *testing.T) {
		bid = uuid.New().String()
		resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 900
				}
               }`)
		resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 930
				}
               }`)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

	})

}
