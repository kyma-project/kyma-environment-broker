package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The blocklist test YAML (testdata/plans-blocklist.yaml) configures:
//   - aws:                provision blocked
//   - azure:              update blocked
//   - build-runtime-gcp:  planUpgrade (target) blocked; gcp is upgradable to build-runtime-gcp

func fixBlocklistConfig() *Config {
	cfg := fixConfig()
	cfg.PlansConfigurationFilePath = "testdata/plans-blocklist.yaml"
	return cfg
}

func TestProvision_BlockedByOperationBlocklist(t *testing.T) {
	cfg := fixBlocklistConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id":    "361c511f-f939-4621-b228-d0fb79a1fe15",
			"context": {
				"globalaccount_id": "g-account-id",
				"subaccount_id":    "sub-id",
				"user_id":          "john.smith@email.com"
			},
			"parameters": {
				"name":   "testing-cluster",
				"region": "eu-central-1"
			}
		}`)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResp := suite.DecodeErrorResponse(resp)
	assert.Contains(t, errResp.Description, "aws provisioning is blocked")
}

func TestUpdate_BlockedByOperationBlocklist(t *testing.T) {
	cfg := fixBlocklistConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	// provision GCP (not blocked)
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id":    "ca6e5357-707f-4565-bbbd-b3ab732597c6",
			"context": {
				"globalaccount_id": "g-account-id",
				"subaccount_id":    "sub-id",
				"user_id":          "john.smith@email.com"
			},
			"parameters": {
				"name":   "testing-cluster",
				"region": "europe-west3"
			}
		}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// switch the instance plan to azure (which has update blocked)
	instance, err := suite.db.Instances().GetByID(iid)
	require.NoError(t, err)
	instance.ServicePlanID = broker.AzurePlanID
	instance.ServicePlanName = broker.AzurePlanName
	instance.Parameters.PlanID = broker.AzurePlanID
	_, err = suite.db.Instances().Update(*instance)
	require.NoError(t, err)

	// attempt update — should be blocked
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id":    "4deee563-e5ec-4731-b9b1-53b42d855f0c",
			"context":    {},
			"parameters": {
				"oidc": {
					"clientID":    "id-ooo",
					"signingAlgs": ["RS256"],
					"issuerURL":   "https://issuer.url.com"
				}
			}
		}`)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResp := suite.DecodeErrorResponse(resp)
	assert.Contains(t, errResp.Description, "azure update is blocked")
}

func TestPlanUpgrade_ToBlockedTargetPlan_IsRejected(t *testing.T) {
	cfg := fixBlocklistConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	// provision GCP (not blocked, and upgradable to build-runtime-gcp)
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id":    "ca6e5357-707f-4565-bbbd-b3ab732597c6",
			"context": {
				"globalaccount_id": "g-account-id",
				"subaccount_id":    "sub-id",
				"user_id":          "john.smith@email.com"
			},
			"parameters": {
				"name":   "testing-cluster",
				"region": "europe-west3"
			}
		}`)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// attempt upgrade gcp -> build-runtime-gcp (target has planUpgrade blocked)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id":    "a310cd6b-6452-45a0-935d-d24ab53f9eba",
			"context":    {},
			"parameters": {}
		}`)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResp := suite.DecodeErrorResponse(resp)
	assert.Contains(t, errResp.Description, "upgrading to build-runtime-gcp is blocked")
}
