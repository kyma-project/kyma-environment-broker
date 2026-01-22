package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	cfg := fixConfig()
	cfg.StepTimeouts.CheckRuntimeResourceCreate = cfg.StepTimeouts.CheckRuntimeResourceCreate / 1000 // reduce timeout to speed up the test
	suite := NewBrokerSuitTestWithMetrics(t, cfg)
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
		defer func() { _ = resp.Body.Close() }()
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
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}

	deleteReq := func(iid string) string {
		resp := suite.CallAPI(
			"DELETE", fmt.Sprintf(
				"oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid), ``)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}

	bindReq := func(iid, bid string) {
		resp := suite.CallAPI(
			"PUT", fmt.Sprintf("oauth/v2/service_instances/%s/service_bindings/%s", iid, bid), `
{
                "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
                "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				"parameters": {
					"expiration_seconds": 600
				}
               }
`)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	t.Run("AssertCorrectMetricValue", func(t *testing.T) {

		// Provisioning

		instance1 := uuid.New().String()
		opID := provisionReq(instance1, broker.AzurePlanID)
		suite.processKIMProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		op1 := suite.GetOperation(opID)
		assert.NotNil(t, op1)

		instance2 := uuid.New().String()
		opID = provisionReq(instance2, broker.TrialPlanID)
		// just wait for timeout, so the operation will fail
		suite.WaitForOperationState(opID, domain.Failed)
		op2 := suite.GetOperation(opID)
		assert.NotNil(t, op2)

		instance3 := uuid.New().String()
		opID = provisionReq(instance3, broker.AWSPlanID)
		suite.processKIMProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		op3 := suite.GetOperation(opID)
		assert.NotNil(t, op3)

		instance4 := uuid.New().String()
		opID = provisionReq(instance4, broker.AzurePlanID)
		suite.failRuntimeByKIM(instance4)
		// should fail when timeout is reached
		suite.WaitForOperationState(opID, domain.Failed)
		op4 := suite.GetOperation(opID)
		assert.NotNil(t, op4)

		instance5 := uuid.New().String()
		opID = provisionReq(instance5, broker.AzurePlanID)
		suite.processKIMProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		op5 := suite.GetOperation(opID)
		assert.NotNil(t, op5)

		instance6 := uuid.New().String()
		opID = provisionReq(instance6, broker.TrialPlanID)
		suite.processKIMProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		op6 := suite.GetOperation(opID)
		assert.NotNil(t, op6)

		instance7 := uuid.New().String()
		opID = provisionReq(instance7, broker.AzurePlanID)
		suite.processKIMProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		op7 := suite.GetOperation(opID)
		assert.NotNil(t, op7)

		// Bindings
		bindReq(instance3, "binding1")
		bindReq(instance3, "binding2")

		// Updates

		opID = updateReq(instance5)
		suite.WaitForOperationState(opID, domain.Succeeded)
		op8 := suite.GetOperation(opID)
		assert.NotNil(t, op8)

		// Deprovisioning

		opID = deleteReq(instance1)
		suite.FinishDeprovisioningOperationByKIM(opID)
		suite.WaitForInstanceArchivedCreated(instance1)
		suite.WaitFor(
			func() bool {
				resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", instance1), ``)
				defer func() { _ = resp.Body.Close() }()
				data := suite.ParseLastOperationResponse(resp)
				return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
			})
		suite.WaitForOperationsNotExists(instance1)
		op9 := suite.GetOperation(opID)
		assert.Nil(t, op9)

		opID = deleteReq(instance7)
		suite.FinishDeprovisioningOperationByKIM(opID)
		suite.WaitForInstanceArchivedCreated(instance7)
		suite.WaitFor(
			func() bool {
				resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", instance7), ``)
				defer func() { _ = resp.Body.Close() }()
				data := suite.ParseLastOperationResponse(resp)
				return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
			})
		suite.WaitForOperationsNotExists(instance7)
		op10 := suite.GetOperation(opID)
		assert.Nil(t, op10)

		opID = deleteReq(instance6)
		suite.WaitForOperationState(opID, domain.Failed)
		op11 := suite.GetOperation(opID)
		assert.NotNil(t, op11)

		opID = deleteReq(instance3)
		suite.WaitForOperationState(opID, domain.Failed)
		op12 := suite.GetOperation(opID)
		assert.NotNil(t, op12)

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

		suite.AssertMetrics2(1, *op1)
		suite.AssertMetrics2(1, *op2)
		suite.AssertMetrics2(1, *op3)
		suite.AssertMetrics2(1, *op4)
		suite.AssertMetrics2(1, *op5)
		suite.AssertMetrics2(1, *op6)
		suite.AssertMetrics2(1, *op7)
		suite.AssertMetrics2(1, *op8)
		suite.AssertMetrics2(1, *op11)

		// uncomment to see the output of the metric endpoint
		//resp := suite.CallAPI("GET", "/metrics", "")
		//r, _ := io.ReadAll(resp.Body)
		//fmt.Printf("%s", r)
	})
}
