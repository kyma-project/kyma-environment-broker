package main

import (
	"fmt"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetParameters_ProvisioningWithCustomOidcConfig(t *testing.T) {
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu21/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region": "eu-central-1",
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"],
						"oidc": {				
							"clientID": "client-id-oidc",
							"groupsClaim": "groups",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": [
									"RS256"
							],
							"usernameClaim": "sub",
							"usernamePrefix": "-"
						}
					}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
			"Name": "testing-cluster"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
				"administrators": ["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"],
				"name": "testing-cluster",
				"oidc": {
					"clientID": "client-id-oidc",
					"groupsClaim": "groups",
					"issuerURL": "https://isssuer.url",
					"signingAlgs": ["RS256"],
					"usernameClaim": "sub",
					"usernamePrefix": "-"
				},
				"region": "eu-central-1"
			},
			"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
			"platform_provider": "Azure",
			"platform_region": "cf-eu21",
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
		},
		"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	}`, iid), string(r))
}

func TestGetParameters_ProvisioningWithNoOidcConfig(t *testing.T) {
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu21/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region": "eu-central-1",
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
					}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
			"Name": "testing-cluster"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
				"administrators": ["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"],
				"name": "testing-cluster",
				"region": "eu-central-1"
			},
			"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
			"platform_provider": "Azure",
			"platform_region": "cf-eu21",
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
		},
		"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	}`, iid), string(r))
}

func TestGetParameters_ProvisioningWithListOidcConfig(t *testing.T) {
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu21/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region": "eu-central-1",
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"],
						"oidc": {
							"list": [
								{
									"clientID": "client-id-oidc",
									"groupsClaim": "groups",
									"issuerURL": "https://isssuer.url",
									"signingAlgs": ["RS256"],
									"usernameClaim": "sub",
									"usernamePrefix": "-"
								}
							]
						}
					}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
			"Name": "testing-cluster"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
				"administrators": ["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"],
				"name": "testing-cluster",
				"oidc": {
					"list": [
						{
							"clientID": "client-id-oidc",
							"groupsClaim": "groups",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": ["RS256"],
							"usernameClaim": "sub",
							"usernamePrefix": "-"
						}
					]
				},
				"region": "eu-central-1"
			},
			"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
			"platform_provider": "Azure",
			"platform_region": "cf-eu21",
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
		},
		"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	}`, iid), string(r))
}
