package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

const deprovisioningRequestPathFormat = "oauth/v2/service_instances/%s?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=%s"

func TestKymaReDeprovisionFailed(t *testing.T) {
	// given
	runtimeOptions := RuntimeOptions{
		GlobalAccountID: globalAccountID,
		SubAccountID:    badSubAccountID,
		Provider:        internal.AWS,
	}

	suite := NewDeprovisioningSuite(t)
	instanceId := suite.CreateProvisionedRuntime(runtimeOptions)
	// when
	deprovisioningOperationID := suite.CreateDeprovisioning(deprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(deprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceNotRemoved(instanceId)

	// when
	reDeprovisioningOperationID := suite.CreateDeprovisioning(reDeprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(reDeprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(reDeprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(reDeprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(reDeprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceNotRemoved(instanceId)
}

func TestReDeprovision(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.EDP.Disabled = true // disable EDP to have all steps successful executed
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
	suite.processProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	// FIRST DEPROVISION
	suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleteError)
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID := suite.DecodeOperationID(resp)
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningID)

	suite.WaitForOperationState(deprovisioningID, domain.Succeeded)

	// SECOND DEPROVISION
	suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleted)
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID = suite.DecodeOperationID(resp)
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningID)
	// then
	suite.WaitForInstanceArchivedCreated(iid)
	suite.WaitFor(func() bool {
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), ``)
		defer resp.Body.Close()
		data := suite.ParseLastOperationResponse(resp)
		return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
	})
	suite.WaitForOperationsNotExists(iid)
}

func TestDeprovisioning_HappyPathAWS(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.EDP.Disabled = true // disable EDP to have all steps successful executed
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

	suite.processProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleted)
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	deprovisioningID := suite.WaitForLastOperation(iid, domain.InProgress)
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningID)

	// then
	suite.WaitForInstanceArchivedCreated(iid)
	suite.WaitFor(func() bool {
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), ``)
		defer resp.Body.Close()
		data := suite.ParseLastOperationResponse(resp)
		return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
	})
	suite.WaitForOperationsNotExists(iid)

}
