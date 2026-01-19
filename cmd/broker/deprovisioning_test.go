package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
)

const deprovisioningRequestPathFormat = "oauth/v2/service_instances/%s?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=%s"

func TestReDeprovision(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	// PROVISION
	// when
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

	suite.processKIMProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	//then
	op, err := suite.db.Operations().GetOperationByID(opID)
	require.NoError(t, err)
	assert.Equal(t, "eu-central-1", op.Region)
	assert.Equal(t, "g-account-id", op.GlobalAccountID)

	// FIRST DEPROVISION
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(deprovisioningID, domain.Failed)

	// SECOND DEPROVISION
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID = suite.DecodeOperationID(resp)

	suite.WaitForOperationState(deprovisioningID, domain.InProgress)
	assert.Equal(t, deprovisioningID, suite.LastOperation(iid).ID)

	//then
	deprovisioningOp, err := suite.db.Operations().GetOperationByID(deprovisioningID)
	require.NoError(t, err)
	assert.Equal(t, "eu-central-1", deprovisioningOp.Region)
	assert.Equal(t, "g-account-id", deprovisioningOp.GlobalAccountID)

	suite.FinishDeprovisioningOperationByKIM(deprovisioningID)
	// then
	suite.WaitForInstanceArchivedCreated(iid)
	suite.WaitFor(func() bool {
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), ``)
		defer func() { _ = resp.Body.Close() }()
		data := suite.ParseLastOperationResponse(resp)
		return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
	})
	suite.WaitForOperationsNotExists(iid)
}

func TestReDeprovisionAlicloud(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	// PROVISION
	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu40/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "9f2c3b4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
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

	suite.processKIMProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	//then
	op, err := suite.db.Operations().GetOperationByID(opID)
	require.NoError(t, err)
	assert.Equal(t, "eu-central-1", op.Region)
	assert.Equal(t, "g-account-id", op.GlobalAccountID)

	// FIRST DEPROVISION
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=9f2c3b4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(deprovisioningID, domain.Failed)

	// SECOND DEPROVISION
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=9f2c3b4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID = suite.DecodeOperationID(resp)

	suite.WaitForOperationState(deprovisioningID, domain.InProgress)
	assert.Equal(t, deprovisioningID, suite.LastOperation(iid).ID)

	//then
	deprovisioningOp, err := suite.db.Operations().GetOperationByID(deprovisioningID)
	require.NoError(t, err)
	assert.Equal(t, "eu-central-1", deprovisioningOp.Region)
	assert.Equal(t, "g-account-id", deprovisioningOp.GlobalAccountID)

	suite.FinishDeprovisioningOperationByKIM(deprovisioningID)
	// then
	suite.WaitForInstanceArchivedCreated(iid)
	suite.WaitFor(func() bool {
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), ``)
		defer func() { _ = resp.Body.Close() }()
		data := suite.ParseLastOperationResponse(resp)
		return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
	})
	suite.WaitForOperationsNotExists(iid)
}
func TestDeprovisioning_HappyPathAWS(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
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

	suite.processKIMProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	opID = suite.DecodeOperationID(resp)
	suite.FinishDeprovisioningOperationByKIM(opID)

	// then
	suite.WaitForInstanceArchivedCreated(iid)
	suite.WaitFor(func() bool {
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), ``)
		defer func() { _ = resp.Body.Close() }()
		data := suite.ParseLastOperationResponse(resp)
		return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
	})
	suite.WaitForOperationsNotExists(iid)

}

func TestRuntimesEndpointForDeprovisionedInstance(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid1 := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid1),
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
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)

	// deprovision
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid1),
		``)
	depOpID := suite.DecodeOperationID(resp)

	suite.FinishDeprovisioningOperationByKIM(depOpID)
	suite.WaitForOperationsNotExists(iid1) // deprovisioning completed, no operations in the DB

	iid2 := uuid.New().String()
	resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=b1a5764e-2ea1-4f95-94c0-2b4538b37b55&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid2),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "b1a5764e-2ea1-4f95-94c0-2b4538b37b55",
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
						"name": "testing-cluster",
						"region": "eu-central-1"
				}
   }`)
	opID = suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)

	// deprovision
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=b1a5764e-2ea1-4f95-94c0-2b4538b37b55&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid2),
		``)
	depOpID = suite.DecodeOperationID(resp)

	suite.FinishDeprovisioningOperationByKIM(depOpID)
	suite.WaitForOperationsNotExists(iid2) // deprovisioning completed, no operations in the DB

	// when
	resp = suite.CallAPI("GET", fmt.Sprintf("runtimes?instance_id=%s&state=deprovisioned", iid1), "")

	// then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var runtimes runtime.RuntimesPage
	response, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(response, &runtimes)
	require.NoError(t, err)

	assert.Len(t, runtimes.Data, 1)
	assert.Equal(t, iid1, runtimes.Data[0].InstanceID)

	// when
	resp = suite.CallAPI("GET", fmt.Sprintf("runtimes?account=%s&subaccount=%s&state=deprovisioned", "g-account-id", "sub-id"), "")

	// then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	response, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(response, &runtimes)
	require.NoError(t, err)

	assert.Len(t, runtimes.Data, 2)
	assert.Contains(t, []string{iid1, iid2}, runtimes.Data[0].InstanceID)
	assert.Contains(t, []string{iid1, iid2}, runtimes.Data[1].InstanceID)

	// when
	resp = suite.CallAPI("GET", fmt.Sprintf("runtimes?plan=%s&state=deprovisioned", "trial"), "")

	// then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	response, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(response, &runtimes)
	require.NoError(t, err)

	assert.Len(t, runtimes.Data, 1)
	assert.Equal(t, iid1, runtimes.Data[0].InstanceID)

	// when
	resp = suite.CallAPI("GET", fmt.Sprintf("runtimes?region=%s&state=deprovisioned", "eu-central-1"), "")

	// then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	response, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(response, &runtimes)
	require.NoError(t, err)

	assert.Len(t, runtimes.Data, 1)
	assert.Equal(t, iid2, runtimes.Data[0].InstanceID)
}
