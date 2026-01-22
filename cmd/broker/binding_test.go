package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v12/domain"
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

	resp := suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	defer func() { _ = resp.Body.Close() }()
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
		`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
	defer func() { _ = resp.Body.Close() }()

	b, _ := io.ReadAll(resp.Body)
	suite.Log(string(b))
	suite.Log(resp.Status)

	t.Run("should return 400 when expiration seconds parameter is string instead of int", func(t *testing.T) {
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": "600"
				}
               }`)
		defer func() { _ = resp.Body.Close() }()
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
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }`)
		defer func() { _ = resp.Body.Close() }()
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }`)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// "expiration_seconds": 600 is a default value from the config in tests
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should return 200 when creating a second binding with the same id and params as an existing one", func(t *testing.T) {
		bid = uuid.New().String()
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }`)
		defer func() { _ = resp.Body.Close() }()

		time.Sleep(2 * time.Second)

		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }`)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		r, _ := io.ReadAll(resp.Body)
		fmt.Printf("%s", r)
	})

	t.Run("should return 409 when creating a second binding with the same id as an existing one but different params", func(t *testing.T) {
		bid = uuid.New().String()
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 900
				}
               }`)
		defer func() { _ = resp.Body.Close() }()
		resp = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
			`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 930
				}
               }`)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

	})

}

func TestDeprovisioningWithExistingBindings(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	bindingID1 := uuid.New().String()
	bindingID2 := uuid.New().String()

	response := suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	opID := suite.DecodeOperationID(response)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when we create two bindings
	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bindingID1),
		`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusCreated, response.StatusCode)

	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bindingID2),
		`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusCreated, response.StatusCode)

	// when we deprovision successfully
	response = suite.CallAPI(http.MethodDelete, fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	defer func() { _ = response.Body.Close() }()
	deprovisioningID := suite.DecodeOperationID(response)
	suite.FinishDeprovisioningOperationByKIM(deprovisioningID)
	suite.WaitForInstanceRemoval(iid)

	// when we remove bindings and the instance is already removed
	response = suite.CallAPI(http.MethodDelete, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s?plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid, bindingID1), "")
	defer func() { _ = response.Body.Close() }()
	assert.Equal(t, http.StatusGone, response.StatusCode)
	response = suite.CallAPI(http.MethodDelete, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s?plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid, bindingID2), "")
	defer func() { _ = response.Body.Close() }()
	assert.Equal(t, http.StatusGone, response.StatusCode)

	// then expect bindings to be removed
	suite.AssertBindingRemoval(iid, bindingID1)
	suite.AssertBindingRemoval(iid, bindingID2)
}

func TestFailedProvisioning(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.StepTimeouts.CheckRuntimeResourceCreate = cfg.StepTimeouts.CheckRuntimeResourceCreate / testSuiteSpeedUpFactor
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	bindingID1 := uuid.New().String()

	response := suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	defer func() { _ = response.Body.Close() }()
	opID := suite.DecodeOperationID(response)
	suite.WaitForOperationState(opID, domain.InProgress)
	// just wait for timeout
	suite.WaitForOperationState(opID, domain.Failed)

	// when we create binding
	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bindingID1),
		`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
	defer func() { _ = response.Body.Close() }()

	// then expect 400 as agreed in the contract
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestProvisioningInProgress(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	bindingID1 := uuid.New().String()

	response := suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	defer func() { _ = response.Body.Close() }()
	opID := suite.DecodeOperationID(response)
	suite.WaitForProvisioningState(opID, domain.InProgress)
	// when we create binding
	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bindingID1),
		`{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15"
               }`)
	defer func() { _ = response.Body.Close() }()

	// then expect 400 as agreed in the contract
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestRemoveBindingsFromSuspended(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.Broker.Binding.BindablePlans = []string{"trial"}
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	bindingID1 := uuid.New().String()
	bindingID2 := uuid.New().String()

	response := suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)
	defer func() { _ = response.Body.Close() }()
	opID := suite.DecodeOperationID(response)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	//then we create two bindings
	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bindingID1),
		`{
	           "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	           "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	          }`)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusCreated, response.StatusCode)

	response = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bindingID2),
		`{
	           "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	           "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	          }`)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusCreated, response.StatusCode)

	//when we suspend Service instance - OSB context update
	response = suite.CallAPI(http.MethodPatch, fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
	   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	   "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	   "context": {
	       "globalaccount_id": "g-account-id",
	       "user_id": "john.smith@email.com",
	       "active": false
	   }
	}`)
	defer func() { _ = response.Body.Close() }()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByKIM(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// when we remove bindings we just return OK
	response = suite.CallAPI(http.MethodDelete, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s?plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid, bindingID1), "")
	defer func() { _ = response.Body.Close() }()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	response = suite.CallAPI(http.MethodDelete, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s?plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid, bindingID2), "")
	defer func() { _ = response.Body.Close() }()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestBindingCreationTimeout(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.Broker.Binding.CreateBindingTimeout = 1 * time.Nanosecond
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	bid := uuid.New().String()

	resp := suite.CallAPI(http.MethodPut, fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"parameters": {}
               }`)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
