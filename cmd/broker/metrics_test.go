package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	cfg := fixConfig()
	cfg.EDP.Disabled = true
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()

	provisionReq := func(iid, plan string) string {
		region := ""
		switch plan {
		case broker.TrialPlanID:
			region = "europe"
		case broker.AzurePlanID:
			region = "westeurope"
		case broker.AWSPlanID:
			region = "eu-central-1"
		}

		body := fmt.Sprintf(`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "%s",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "%s"
					}
		}`, plan, region)
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid), body)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}

	updateReq := func(iid string) string {
		resp := suite.CallAPI(
			"PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"context": {
				"license_type": "CUSTOMER"
			}
		}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}

	deleteReq := func(iid string) string {
		resp := suite.CallAPI(
			"DELETE", fmt.Sprintf(
				"oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid), ``)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}

	t.Run("AssertCorrectMetricValue", func(t *testing.T) {

		// Provisioning

		instance1 := uuid.New().String()
		opID := provisionReq(instance1, broker.AzurePlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		instance2 := uuid.New().String()
		opID = provisionReq(instance2, broker.TrialPlanID)
		suite.failProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Failed)

		instance3 := uuid.New().String()
		opID = provisionReq(instance3, broker.AWSPlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		instance4 := uuid.New().String()
		opID = provisionReq(instance4, broker.AzurePlanID)
		suite.failProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Failed)

		instance5 := uuid.New().String()
		opID = provisionReq(instance5, broker.AzurePlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		instance6 := uuid.New().String()
		opID = provisionReq(instance6, broker.TrialPlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		instance7 := uuid.New().String()
		opID = provisionReq(instance7, broker.AzurePlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// Updates

		opID = updateReq(instance5)
		suite.FinishUpdatingOperationByProvisioner(opID)
		suite.FinishUpdatingOperationByReconciler(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// Deprovisioning

		suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleted)
		opID = deleteReq(instance1)
		suite.FinishDeprovisioningByReconciler(opID)
		suite.FinishDeprovisioningOperationByProvisioner(opID)
		suite.WaitForInstanceArchivedCreated(instance1)
		suite.WaitFor(
			func() bool {
				resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", instance1), ``)
				defer resp.Body.Close()
				data := suite.ParseLastOperationResponse(resp)
				return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
			})
		suite.WaitForOperationsNotExists(instance1)

		suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleted)
		opID = deleteReq(instance7)
		suite.FinishDeprovisioningByReconciler(opID)
		suite.FinishDeprovisioningOperationByProvisioner(opID)
		suite.WaitForInstanceArchivedCreated(instance7)
		suite.WaitFor(
			func() bool {
				resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", instance7), ``)
				defer resp.Body.Close()
				data := suite.ParseLastOperationResponse(resp)
				return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
			})
		suite.WaitForOperationsNotExists(instance7)

		suite.SetReconcilerResponseStatus(reconcilerApi.StatusError)
		opID = deleteReq(instance6)
		suite.FailDeprovisioningOperationByProvisioner(opID)
		suite.WaitForOperationState(opID, domain.Failed)

		suite.SetReconcilerResponseStatus(reconcilerApi.StatusError)
		opID = deleteReq(instance3)
		suite.FailDeprovisioningOperationByProvisioner(opID)
		suite.WaitForOperationState(opID, domain.Failed)

		time.Sleep(1 * time.Second)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Succeeded, broker.AzurePlanID, 3)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Succeeded, broker.TrialPlanID, 1)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Succeeded, broker.AWSPlanID, 1)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Failed, broker.AzurePlanID, 1)
		suite.AssertMetric(internal.OperationTypeUpdate, domain.Succeeded, broker.AzurePlanID, 1)
		suite.AssertMetric(internal.OperationTypeUpdate, domain.Failed, broker.AzurePlanID, 0)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Succeeded, broker.AzurePlanID, 2)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Succeeded, broker.TrialPlanID, 0)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Failed, broker.AzurePlanID, 0)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Failed, broker.AWSPlanID, 1)
	})
}
