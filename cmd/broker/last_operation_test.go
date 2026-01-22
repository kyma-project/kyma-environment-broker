package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLastOperationWithoutOperationIDHappyPath(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
		"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
		"context": {
			"sm_operator_credentials": {
				"clientid": "cid",
				"clientsecret": "cs",
				"url": "url",
				"sm_url": "sm_url"
			},
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
		},
		"parameters": {
			"name": "testing-cluster"
		}
	}`)
	defer func() { _ = resp.Body.Close() }()
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)

	//when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), "")
	defer func() { _ = resp.Body.Close() }()

	//then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLastOperationWithOperationIDHappyPath(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
		"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
		"context": {
			"sm_operator_credentials": {
				"clientid": "cid",
				"clientsecret": "cs",
				"url": "url",
				"sm_url": "sm_url"
			},
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
		},
		"parameters": {
			"name": "testing-cluster"
		}
	}`)
	defer func() { _ = resp.Body.Close() }()
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)

	//when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation?operation=%s", iid, opID), "")
	defer func() { _ = resp.Body.Close() }()

	//then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLastOperationNotExistingInstance(t *testing.T) {
	//given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	//when
	resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), "")
	defer func() { _ = resp.Body.Close() }()

	//then
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)
	assert.Contains(t, errResponse.Description, "Operation not found")
}

func TestLastOperationNotExistingOperation(t *testing.T) {
	//given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()
	oid := uuid.New().String()

	//when
	resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation?operation=%s", iid, oid), "")
	defer func() { _ = resp.Body.Close() }()

	//then
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)
	assert.Contains(t, errResponse.Description, "Operation not found")
}

func TestLastOperationWithOperationIDAndNotExistingInstanceID(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"sm_operator_credentials": {
					"clientid": "cid",
					"clientsecret": "cs",
					"url": "url",
					"sm_url": "sm_url"
				},
				"globalaccount_id": "g-account-id",
				"subaccount_id": "sub-id",
				"user_id": "john.smith@email.com"
			},
			"parameters": {
				"name": "testing-cluster"
			}
		}`)
	defer func() { _ = resp.Body.Close() }()
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)

	oid := uuid.New().String()

	//when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation?operation=%s", oid, opID), "")
	defer func() { _ = resp.Body.Close() }()

	//then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
