package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/customresources"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/google/uuid"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const updateRequestPathFormat = "oauth/v2/service_instances/%s?accepts_incomplete=true"

func TestUpdate(t *testing.T) {
	// given
	cfg := fixConfig()
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
						"name": "testing-cluster",
						"oidc": {
							"clientID": "id-initial",
							"signingAlgs": ["PS512"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	assert.Equal(t, opID, suite.LastOperation(iid).ID)
	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RS256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	assert.Equal(t, upgradeOperationID, suite.LastOperation(iid).ID)

	op, err := suite.db.Operations().GetOperationByID(upgradeOperationID)
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", op.Region)
	assert.Equal(t, "g-account-id", op.GlobalAccountID)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	oidc := (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0]

	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"RS256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-west-1",
		"kyma-project.io/platform-region": "cf-eu10",
	})
}

func TestUpdateWithKIM(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
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
						"oidc": {
							"clientID": "id-initial",
							"signingAlgs": ["PS512"],
                            "issuerURL": "https://issuer.url.com"
						},
						"region": "eu-central-1"
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.waitForRuntimeAndMakeItReady(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RS256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(iid)

	assert.Equal(t, "id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
}

func TestUpdatePlan(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
	opID := suite.DecodeOperationID(resp)
	suite.waitForRuntimeAndMakeItReady(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "6aae0ff3-89f7-4f12-86de-51466145422e",
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
			}
   }`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	gotInstance := suite.GetInstance(iid)
	assert.Equal(t, "6aae0ff3-89f7-4f12-86de-51466145422e", gotInstance.ServicePlanID)
	assert.Equal(t, "6aae0ff3-89f7-4f12-86de-51466145422e", gotInstance.Parameters.PlanID)
	assert.Equal(t, "build-runtime-aws", gotInstance.ServicePlanName)

	updateOperation := suite.GetOperation(updateOperationID)
	assert.Equal(t, "6aae0ff3-89f7-4f12-86de-51466145422e", updateOperation.ProvisioningParameters.PlanID)

	suite.AssertRuntimeResourceLabels(updateOperationID)
	suite.AssertKymaLabelsExist(updateOperationID, map[string]string{
		customresources.PlanIdLabel:   "6aae0ff3-89f7-4f12-86de-51466145422e",
		customresources.PlanNameLabel: "build-runtime-aws",
	})

	actions, err := suite.db.Actions().ListActionsByInstanceID(iid)
	assert.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, actions[0].Type, pkg.PlanUpdateActionType)
	assert.Equal(t, actions[0].Message, "Plan updated from aws (PlanID: 361c511f-f939-4621-b228-d0fb79a1fe15) to build-runtime-aws (PlanID: 6aae0ff3-89f7-4f12-86de-51466145422e).")
	assert.Equal(t, actions[0].OldValue, "361c511f-f939-4621-b228-d0fb79a1fe15")
	assert.Equal(t, actions[0].NewValue, "6aae0ff3-89f7-4f12-86de-51466145422e")
}

func TestUpdateFailedInstance(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.StepTimeouts.CheckRuntimeResourceCreate = cfg.StepTimeouts.CheckRuntimeResourceCreate / testSuiteSpeedUpFactor

	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	// just wait for timeout and failed operation
	suite.WaitForOperationState(opID, domain.Failed)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RSA256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)

	assert.Equal(t, "Unable to process an update of a failed instance", errResponse.Description)
}

func TestUpdate_SapConvergedCloud(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu20/v2/service_instances/%s?accepts_incomplete=true&plan_id=03b812ac-c991-4528-b5bd-08b303523a63&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "03b812ac-c991-4528-b5bd-08b303523a63",
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
						"oidc": {
							"clientID": "id-initial",
							"signingAlgs": ["PS512"],
                            "issuerURL": "https://issuer.url.com"
						},
						"region": "eu-de-1"

			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu20/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "03b812ac-c991-4528-b5bd-08b303523a63",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RS256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "id-ooo", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"RS256"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "groups", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-de-1",
		"kyma-project.io/platform-region": "cf-eu20",
	})
}

func TestUpdateDeprovisioningInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	depOpID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(depOpID, domain.InProgress)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RSA256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)

	assert.Equal(t, "Unable to process an update of a deprovisioned instance", errResponse.Description)
}

func TestUpdateWithNoOIDCParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, defaultOIDCValues().ClientID, *(*oidc)[0].ClientID)
	assert.Equal(t, defaultOIDCValues().SigningAlgs, (*oidc)[0].SigningAlgs)
	assert.Equal(t, defaultOIDCValues().IssuerURL, *(*oidc)[0].IssuerURL)
	assert.Equal(t, defaultOIDCValues().GroupsClaim, *(*oidc)[0].GroupsClaim)
	assert.Equal(t, defaultOIDCValues().UsernameClaim, *(*oidc)[0].UsernameClaim)
	assert.Equal(t, defaultOIDCValues().UsernamePrefix, *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-west-1",
		"kyma-project.io/platform-region": "cf-eu10",
	})

}

func TestUpdateWithNoOidcOnUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"signingAlgs": ["RS256"],
					"issuerURL": "https://issuer.url.com"
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "id-ooo", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"RS256"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "groups", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)
}

func TestUpdateContext(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"signingAlgs": ["RS384"],
					"issuerURL": "https://issuer.url.com"
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	// OSB update
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-west-1",
		"kyma-project.io/platform-region": "cf-eu10",
	})

}

func TestKymaResourceNameAndGardenerClusterNameAfterUnsuspension(t *testing.T) {
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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

	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.Log("*** Suspension ***")

	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByKIM(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// OSB update
	suite.Log("*** Unsuspension ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
			"active": true
       }
       
   }`)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suite.processKIMProvisioningByInstanceID(iid)

	// the old Kyma resource not exists
	suite.AssertKymaResourceNotExists(opID)
	instance := suite.GetInstance(iid)
	assert.Equal(t, instance.RuntimeID, instance.InstanceDetails.KymaResourceName)
	//time.Sleep(time.Second)
	suite.AssertKymaResourceExistsByInstanceID(iid)
}

func TestUpdateWithOwnClusterPlan(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
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
						"shootName": "shoot-name",
						"shootDomain": "kyma-dev.shoot.canary.k8s-hana.ondemand.com",
						"kubeconfig":"YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmN1cnJlbnQtY29udGV4dDogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKY29udGV4dHM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY29udGV4dDoKICAgICAgY2x1c3Rlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKICAgICAgdXNlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUtdG9rZW4KY2x1c3RlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2FwaS5jbHVzdGVyLW5hbWUua3ltYS1kZXYuc2hvb3QuY2FuYXJ5Lms4cy1oYW5hLm9uZGVtYW5kLmNvbQogICAgICBjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YTogPi0KICAgICAgICBMUzB0TFMxQ1JVZEpUaUJEUlZKVVNVWkpRMEZVUlMwdExTMHQKdXNlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZS10b2tlbgogICAgdXNlcjoKICAgICAgdG9rZW46ID4tCiAgICAgICAgdE9rRW4K"
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"kubeconfig":"YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmN1cnJlbnQtY29udGV4dDogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKY29udGV4dHM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY29udGV4dDoKICAgICAgY2x1c3Rlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKICAgICAgdXNlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUtdG9rZW4KY2x1c3RlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2FwaS5jbHVzdGVyLW5hbWUua3ltYS1kZXYuc2hvb3QuY2FuYXJ5Lms4cy1oYW5hLm9uZGVtYW5kLmNvbQogICAgICBjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YTogPi0KICAgICAgICBMUzB0TFMxQ1JVZEpUaUJEUlZKVVNVWkpRMEZVUlMwdExTMHQKdXNlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZS10b2tlbgogICAgdXNlcjoKICAgICAgdG9rZW46ID4tCiAgICAgICAgdE9rRW4K"
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.AssertKymaResourceExists(upgradeOperationID)
	suite.AssertKymaLabelsExist(upgradeOperationID, map[string]string{
		"kyma-project.io/platform-region": "cf-eu10",
	})
}

func TestUpdateOidcForSuspendedInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"signingAlgs": ["RS256"],
					"issuerURL": "https://issuer.url.com"
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.Log("*** Suspension ***")

	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByKIM(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// WHEN
	// OSB update
	suite.Log("*** Update ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
       "parameters": {
       		"oidc": {
				"clientID": "id-oooxx",
				"signingAlgs": ["RS256"],
                "issuerURL": "https://issuer.url.com"
			}
       }
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOpID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOpID, domain.Succeeded)

	// THEN
	instance := suite.GetInstance(iid)
	assert.Equal(t, "id-oooxx", instance.Parameters.Parameters.OIDC.ClientID)

	// Start unsuspension
	// OSB update (unsuspension)
	suite.Log("*** Update (unsuspension) ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": true
       }
   }`)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// WHEN
	suite.processKIMProvisioningByInstanceID(iid)

	// THEN
	suite.WaitForLastOperation(iid, domain.Succeeded)
	instance = suite.GetInstance(iid)
	assert.Equal(t, "id-oooxx", instance.Parameters.Parameters.OIDC.ClientID)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	assert.Equal(t, "id-oooxx", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)

	suite.AssertKymaResourceNotExists(opID)
	suite.AssertKymaResourceExistsByInstanceID(iid)
}

func TestUpdateNotExistingInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"signingAlgs": ["RS256"],
					"issuerURL": "https://issuer.url.com"
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// provisioning done, let's start an update

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/not-existing"),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       }
   }`)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-west-1",
		"kyma-project.io/platform-region": "cf-eu10",
	})
}

func TestUpdateDefaultAdminNotChanged(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
      "context": {
          "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
      },
		"parameters": {
		}
  }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(id)

	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)

}

func TestUpdateDefaultAdminNotChangedWithCustomOIDC(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"issuerURL": "https://issuer.url.com"
				}
			}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(id)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "id-ooo", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"RS256"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "groups", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)
}

func TestUpdateDefaultAdminNotChangedWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RS384"],
				"issuerURL": "https://issuer.url.com",
				"groupsClaim": "new-groups-claim",
				"usernameClaim": "new-username-claim",
				"usernamePrefix": "->"
			}
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(id)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "id-ooo", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"RS384"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "new-groups-claim", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "new-username-claim", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "->", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)
}

func TestUpdateDefaultAdminOverwritten(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin1@kyma.cx", "newAdmin2@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(id)
	assert.Equal(t, expectedAdmins, runtime.Spec.Security.Administrators)
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsNotChanged(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin1@kyma.cx", "newAdmin2@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
				"name": "testing-cluster",
				"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	r := suite.GetRuntimeResourceByInstanceID(id)

	fmt.Println("Runtime: ", r.Spec.Security.Administrators)
	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "jack.anvil@email.com"
       },
		"parameters": {
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	time.Sleep(time.Second)
	runtime := suite.GetRuntimeResourceByInstanceID(id)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "client-id-oidc", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"RS256"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "groups", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, expectedAdmins, runtime.Spec.Security.Administrators)
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsNotChangedWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin1@kyma.cx", "newAdmin2@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
				"name": "testing-cluster",
				"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["ES256"],
				"issuerURL": "https://newissuer.url.com"
			}
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(id)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "id-ooo", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"ES256"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://newissuer.url.com", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "groups", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, expectedAdmins, runtime.Spec.Security.Administrators)

	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsOverwritten(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin3@kyma.cx", "newAdmin4@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
				"name": "testing-cluster",
				"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"administrators":["newAdmin3@kyma.cx", "newAdmin4@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(id)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "client-id-oidc", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"RS256"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "groups", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, expectedAdmins, runtime.Spec.Security.Administrators)

	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsOverwrittenWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin3@kyma.cx", "newAdmin4@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
				"name": "testing-cluster",
				"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["ES384"],
				"issuerURL": "https://issuer.url.com",
				"groupsClaim": "new-groups-claim"
			},
			"administrators":["newAdmin3@kyma.cx", "newAdmin4@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(id)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	assert.Equal(t, "id-ooo", *(*oidc)[0].ClientID)
	assert.Equal(t, []string{"ES384"}, (*oidc)[0].SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *(*oidc)[0].IssuerURL)
	assert.Equal(t, "new-groups-claim", *(*oidc)[0].GroupsClaim)
	assert.Equal(t, "sub", *(*oidc)[0].UsernameClaim)
	assert.Equal(t, "-", *(*oidc)[0].UsernamePrefix)
	assert.Equal(t, expectedAdmins, runtime.Spec.Security.Administrators)

	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsOverwrittenTwice(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins1 := []string{"newAdmin3@kyma.cx", "newAdmin4@kyma.cx"}
	expectedAdmins2 := []string{"newAdmin5@kyma.cx", "newAdmin6@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
				"name": "testing-cluster",
				"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"administrators":["newAdmin3@kyma.cx", "newAdmin4@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(id)
	assert.Equal(t, expectedAdmins1, runtime.Spec.Security.Administrators)

	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins1)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["PS256"],
				"issuerURL": "https://newissuer.url.com",
				"usernamePrefix": "->"
			},
			"administrators":["newAdmin5@kyma.cx", "newAdmin6@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID = suite.DecodeOperationID(resp)
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime = suite.GetRuntimeResourceByInstanceID(id)
	assert.Equal(t, expectedAdmins2, runtime.Spec.Security.Administrators)
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins2)
}

func TestUpdateAutoscalerParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
		"region":"eu-central-1",
		"name": "testing-cluster",
		"autoScalerMin":5,
		"autoScalerMax":7,
		"maxSurge":3,
		"maxUnavailable":4
	}
}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":15,
		"autoScalerMax":25,
		"maxSurge":10,
		"maxUnavailable":7
	}
}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	min, max, surge, unav := 15, 25, 10, 7
	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime := suite.GetRuntimeResourceByInstanceID(id)
	assert.True(t, runtime.Spec.Security.Networking.Filter.Egress.Enabled)
	assert.Equal(t, min, int(runtime.Spec.Shoot.Provider.Workers[0].Minimum))
	assert.Equal(t, max, int(runtime.Spec.Shoot.Provider.Workers[0].Maximum))
	assert.Equal(t, surge, runtime.Spec.Shoot.Provider.Workers[0].MaxSurge.IntValue())
	assert.Equal(t, unav, runtime.Spec.Shoot.Provider.Workers[0].MaxUnavailable.IntValue())

	assert.Equal(t, imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("client-id-oidc"),
			GroupsClaim:    ptr.String("groups"),
			IssuerURL:      ptr.String("https://issuer.url"),
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  ptr.String("sub"),
			UsernamePrefix: ptr.String("-"),
			GroupsPrefix:   ptr.String("-"),
		},
	}, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])

	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)
}

func TestUpdateAutoscalerWrongParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
		"region":"eu-central-1",
		"name": "testing-cluster",
		"autoScalerMin":5,
		"autoScalerMax":7,
		"maxSurge":3,
		"maxUnavailable":4
	}
}`)

	opID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, opID)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":26,
		"autoScalerMax":25,
		"maxSurge":10,
		"maxUnavailable":7
	}
}`)

	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateAutoscalerPartialSequence(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
		"region":"eu-central-1",
		"name": "testing-cluster"
	}
}`)

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// when autoScalerMin is updated with value greater than autoScalerMax
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":25
	}
}`)

	// then
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMax":15
	}
}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	assert.NotEmpty(t, upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(id)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)
	assert.Equal(t, 15, int(runtime.Spec.Shoot.Provider.Workers[0].Maximum))

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":14
	}
}`)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID = suite.DecodeOperationID(resp)
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime = suite.GetRuntimeResourceByInstanceID(id)
	assert.Equal(t, 14, int(runtime.Spec.Shoot.Provider.Workers[0].Minimum))

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":16
	}
}`)

	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateWhenBothErsContextAndUpdateParametersProvided(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()

	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"signingAlgs": ["RS256"],
					"issuerURL": "https://issuer.url.com"
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.Log("*** Suspension ***")

	// when
	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       },
       "parameters": {
			"name": "testing-cluster"
		}
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionID := suite.WaitForLastOperation(iid, domain.InProgress)
	suite.FinishDeprovisioningOperationByKIM(suspensionID)

	suite.WaitForLastOperation(iid, domain.Succeeded)

	// THEN
	lastOp, err := suite.db.Operations().GetLastOperation(iid)
	require.NoError(t, err)
	assert.Equal(t, internal.OperationTypeDeprovision, lastOp.Type, "last operation should be type deprovision")

	updateOps, err := suite.db.Operations().ListUpdatingOperationsByInstanceID(iid)
	require.NoError(t, err)
	assert.Len(t, updateOps, 0, "should not create any update operations")
}

func TestUpdateMachineType(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := "InstanceID-machineType"

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	_, err := suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "instance after provisioning")

	// when patch to change machine type

	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "john.smith@email.com"
	},
	"parameters": {
		"machineType":"m5.2xlarge"
	}
}`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(id)
	assert.Equal(t, "m5.2xlarge", runtime.Spec.Shoot.Provider.Workers[0].Machine.Type)

}
func TestUpdateNetworkFilterForExternal(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_operator_credentials": {
							"clientid": "testClientID",
							"clientsecret": "testClientSecret",
							"sm_url": "https://service-manager.kyma.com",
							"url": "https://test.auth.com",
							"xsappname": "testXsappname"
						},
						"license_type": "CUSTOMER",
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
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then

	suite.AssertNetworkFiltering(instance.InstanceID, false, false)
	assert.Equal(suite.t, "CUSTOMER", *instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname"
				}
			},
			"parameters": {
				"name": "testing-cluster"
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	updateOp, _ := suite.db.Operations().GetOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	suite.AssertNetworkFiltering(instance.InstanceID, false, false)
	instance2 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
}

func TestUpdateNetworkFilterForInternal(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
						"sm_operator_credentials": {
							"clientid": "testClientID",
							"clientsecret": "testClientSecret",
							"sm_url": "https://service-manager.kyma.com",
							"url": "https://test.auth.com",
							"xsappname": "testXsappname"
						},
						"license_type": "NON-CUSTOMER",
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
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then

	suite.AssertNetworkFiltering(instance.InstanceID, true, false)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname"
				}
			},
			"parameters": {
				"name": "testing-cluster",
				"ingressFiltering": true
			}
		}`)

	// then
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	updateOp, _ := suite.db.Operations().GetOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	suite.AssertNetworkFiltering(instance.InstanceID, true, true)
	// check if updated parameters is populated to provisioning parameters - so it will be reflected in get instance response
	instanceUpdated := suite.GetInstance(id)
	assert.True(suite.t, *instanceUpdated.Parameters.Parameters.IngressFiltering)
}

func TestUpdateNetworkFilterForExternal_WithIngressForExternal(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_operator_credentials": {
							"clientid": "testClientID",
							"clientsecret": "testClientSecret",
							"sm_url": "https://service-manager.kyma.com",
							"url": "https://test.auth.com",
							"xsappname": "testXsappname"
						},
						"license_type": "CUSTOMER",
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
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then

	suite.AssertNetworkFiltering(instance.InstanceID, false, false)
	assert.Equal(suite.t, "CUSTOMER", *instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname"
				}
			},
			"parameters": {
				"name": "testing-cluster",
				"ingressFiltering": true
			}
		}`)

	// then
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
	parsedResponse := suite.ReadResponse(resp)
	assert.Contains(t, string(parsedResponse), "ingress filtering option is not available")
}

func TestUpdateStoreNetworkFilterAndUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname2"
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
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then
	suite.AssertNetworkFiltering(instance.InstanceID, true, false)
	assert.Nil(suite.t, instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname"
				},
				"license_type": "CUSTOMER"
			}
		}`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	//then
	updateOp, _ := suite.db.Operations().GetOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	instance2 := suite.GetInstance(id)
	// license_type should be stored in the instance table for ERS context and future upgrades
	suite.AssertNetworkFiltering(instance.InstanceID, false, false)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
}

func TestMultipleUpdateNetworkFilterPersisted(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname2"
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
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then
	suite.AssertNetworkFiltering(instance.InstanceID, true, false)
	assert.Nil(suite.t, instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"context": {
				"license_type": "CUSTOMER"
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	instance2 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"context":{},
			"parameters":{
			    "name":"instance",
			    "administrators":["xyz@sap.com", "xyz@gmail.com", "xyz@abc.com"]
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperation2ID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperation2ID, domain.Succeeded)
	instance3 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance3.Parameters.ErsContext.LicenseType)
	// we do not support updating network filtering accordingly when the license type is changed
	suite.AssertNetworkFiltering(instance.InstanceID, false, false)
}

func TestUpdateOnlyErsContextForExpiredInstance(t *testing.T) {
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	response := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	opID := suite.DecodeOperationID(response)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("expire/service_instance/%s", iid), "")
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	response = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
		"context": {
			"globalaccount_id": "g-account-id-new"
		}
	}`)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestUpdateParamsForExpiredInstance(t *testing.T) {
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	response := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	opID := suite.DecodeOperationID(response)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("expire/service_instance/%s", iid), "")
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	response = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"parameters":{
					"administrators":["xyz@sap.com", "xyz@gmail.com", "xyz@abc.com"]
				}
			}`)
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
}

func TestUpdateErsContextAndParamsForExpiredInstance(t *testing.T) {
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()
	response := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	opID := suite.DecodeOperationID(response)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	response = suite.CallAPI(http.MethodPut, fmt.Sprintf("expire/service_instance/%s", iid), "")
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	response = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",	
				"parameters": {
					"administrators":["xyz2@sap.com", "xyz2@gmail.com", "xyz2@abc.com"]
				},
				"context": {
					"license_type": "CUSTOMER"
				}
		}`)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestUpdateAdditionalWorkerNodePools(t *testing.T) {
	t.Run("should add additional worker node pools", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-2",
									"machineType": "m5.large",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		// then
		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)
		assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 2)
		suite.assertAdditionalWorkerIsCreated(t, runtime.Spec.Shoot.Provider, "name-1", "m6i.large", 3, 20, 3)
		suite.assertAdditionalWorkerIsCreated(t, runtime.Spec.Shoot.Provider, "name-2", "m5.large", 4, 21, 1)
	})

	t.Run("should replace additional worker node pools", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								}
							]
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-2",
									"machineType": "m5.large",
									"haZones": true,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		// then
		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)
		assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 1)
		suite.assertAdditionalWorkerIsCreated(t, runtime.Spec.Shoot.Provider, "name-2", "m5.large", 4, 21, 3)
	})

	t.Run("should remove additional worker node pools when list is empty", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": false,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-2",
									"machineType": "m5.large",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": []
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		// then
		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)
		assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 0)
	})

	t.Run("updated additional worker node pool should have the same zones", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"additionalWorkerNodePools": [
								{
									"name": "worker-1",
									"machineType": "m6i.large",
									"haZones": false,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "worker-2",
									"machineType": "m6i.large",
									"haZones": false,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "worker-3",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "worker-4",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								}
							]
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)
		worker1Zones := (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[0].Zones
		worker2Zones := (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[1].Zones
		worker3Zones := (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[2].Zones
		worker4Zones := (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[3].Zones

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "worker-1",
									"machineType": "m6i.large",
									"haZones": false,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "worker-2",
									"machineType": "m6i.large",
									"haZones": false,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "worker-3",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "worker-4",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)
		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		updatedRuntime := suite.GetRuntimeResourceByInstanceID(iid)
		updatedWorker1Zones := (*updatedRuntime.Spec.Shoot.Provider.AdditionalWorkers)[0].Zones
		updatedWorker2Zones := (*updatedRuntime.Spec.Shoot.Provider.AdditionalWorkers)[1].Zones
		updatedWorker3Zones := (*updatedRuntime.Spec.Shoot.Provider.AdditionalWorkers)[2].Zones
		updatedWorker4Zones := (*updatedRuntime.Spec.Shoot.Provider.AdditionalWorkers)[3].Zones

		// then
		assert.Equal(t, worker1Zones, updatedWorker1Zones)
		assert.Equal(t, worker2Zones, updatedWorker2Zones)
		assert.Equal(t, worker3Zones, updatedWorker3Zones)
		assert.Equal(t, worker4Zones, updatedWorker4Zones)
	})

	t.Run("should add additional worker node pools with zones from zone mapping", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"region": "us-east-1"
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "c7i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-2",
									"machineType": "g6.xlarge",
									"haZones": false,
									"autoScalerMin": 1,
									"autoScalerMax": 1
								},
								{
									"name": "name-3",
									"machineType": "g4dn.xlarge",
									"haZones": false,
									"autoScalerMin": 1,
									"autoScalerMax": 1
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		// then
		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)
		assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 3)
		suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-1", 3, "us-east-1w", "us-east-1x", "us-east-1y", "us-east-1z")
		suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-2", 1, "us-east-1x", "us-east-1y")
		suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-3", 1, "us-east-1x")
	})

	t.Run("should update machine type in the additional worker node pool", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"region": "us-east-1",
                            "additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "m5.large",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// update machine type of the additional worker node pool
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "m5.xlarge",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		// then
		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)
		assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 1)
		suite.assertAdditionalWorkerIsCreated(t, runtime.Spec.Shoot.Provider, "name-11", "m5.xlarge", 4, 21, 1)

		// try to update with an additional machine type g6.xlarge
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "g6.xlarge",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("should not update additional worker node pool when initial machine type is not a regular one", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"region": "us-east-1",
                            "additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "g6.xlarge",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// try to update with a regular machine type m5.large
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "m5.large",
									"haZones": false,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("should update additional worker node pool when initial machine type is a regular one", func(t *testing.T) {
		// given
		cfg := fixConfig()

		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"region": "us-east-1",
                            "additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "m5.xlarge",
									"haZones": true,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		// try to update with a regular machine type m5.large
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-11",
									"machineType": "m5.large",
									"haZones": true,
									"autoScalerMin": 4,
									"autoScalerMax": 21
								}
							]
						}
   			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})
}

func TestUpdateOIDC(t *testing.T) {
	t.Run("should update OIDC object with OIDC list", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com"
					},
					"region": "eu-central-1"
				}
			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"list": [
							{
								"clientID": "id-ooo",
								"signingAlgs": ["RS256"],
								"issuerURL": "https://issuer.url.com",
								"groupsClaim": "groups",
                				"groupsPrefix": "-",
								"usernameClaim": "sub",
                				"usernamePrefix": "-"
							},
							{
								"clientID": "id-ooo2",
								"signingAlgs": ["RS256"],
								"issuerURL": "https://issuer.url.com",
								"groupsClaim": "groups",
                				"groupsPrefix": "-",
								"usernameClaim": "sub",
                				"usernamePrefix": "-"
							}
						]
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Equal(t, "id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
		assert.Equal(t, "id-ooo2", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[1].ClientID)
		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 2)
	})
	t.Run("should update OIDC object with OIDC object", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com"
					},
					"region": "eu-central-1"
				}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"clientID": "id-ooo",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com"
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 1)
		assert.Equal(t, "id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
	})
	t.Run("should remove previously set required claims", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com",
						"requiredClaims": ["claim1=value1", "claim2=value2"]
					},
					"region": "eu-central-1"
				}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"clientID": "id-ooo",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com",
						"requiredClaims": ["-"]
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Equal(t, "id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
		assert.Nil(t, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].RequiredClaims)
		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 1)
	})
	t.Run("should reject update OIDC list with OIDC object", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"list": [
							{
								"clientID": "id-ooo",
								"signingAlgs": ["RS256"],	
								"groupsClaim": "fakeGroups",
								"usernameClaim": "fakeUsernameClaim",
								"usernamePrefix": "::",
								"groupsPrefix": "-",
								"issuerURL": "https://issuer.url.com"
							},
							{
								"clientID": "id-ooo2",
								"groupsClaim": "fakeGroups",
								"usernameClaim": "fakeUsernameClaim",
								"usernamePrefix": "::",
								"groupsPrefix": "-",
								"signingAlgs": ["RS256"],
								"issuerURL": "https://issuer.url.com"
							}
						]
					},
					"region": "eu-central-1"
				}
			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"clientID": "id-client",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com"
					}
				}
			}`)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("should reject update empty OIDC list with OIDC object that has no values", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"list": []
					},
					"region": "eu-central-1"
				}
			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"clientID": "",
						"signingAlgs": [],
						"issuerURL": ""
					}
				}
			}`)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("should update OIDC list with OIDC list", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"list": [
							{
								"clientID": "id-ooo",
								"signingAlgs": ["RS256"],	
								"groupsClaim": "fakeGroups",
								"usernameClaim": "fakeUsernameClaim",
								"usernamePrefix": "::",
								"groupsPrefix": "-",
								"issuerURL": "https://issuer.url.com"
							},
							{
								"clientID": "id-ooo2",
								"groupsClaim": "fakeGroups",
								"usernameClaim": "fakeUsernameClaim",
								"usernamePrefix": "::",
								"groupsPrefix": "-",
								"signingAlgs": ["RS256"],
								"issuerURL": "https://issuer.url.com"
							}
						]
					},
					"region": "eu-central-1"
				}
			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"list": [
							{
								"clientID": "new-id-ooo",
								"groupsClaim": "fakeGroups",
								"usernameClaim": "fakeUsernameClaim",
								"usernamePrefix": "::",
								"signingAlgs": ["RS256"],
								"groupsPrefix": "-",
								"issuerURL": "https://issuer.url.com"
							}
						]
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Equal(t, "new-id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 1)
	})
	t.Run("should update OIDC object with empty OIDC list", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)

		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com"
					},
					"region": "eu-central-1"
				}
			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"list": []
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 0)
	})
	t.Run("should remove JWKS from OIDC config", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com",
						"encodedJwksArray": "andrcy10b2tlbi1kZWZhdWx0"
					},
					"region": "eu-central-1"
				}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"clientID": "id-ooo",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com",
						"encodedJwksArray": "-"
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Equal(t, "id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
		assert.Nil(t, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].JWKS)
		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 1)
	})
	t.Run("should not remove JWKS from OIDC config", func(t *testing.T) {
		// given
		cfg := fixConfig()
		suite := NewBrokerSuiteTestWithConfig(t, cfg)
		defer suite.TearDown()
		iid := uuid.New().String()

		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com",
						"encodedJwksArray": "andrcy10b2tlbi1kZWZhdWx0"
					},
					"region": "eu-central-1"
				}
   			}`)
		opID := suite.DecodeOperationID(resp)
		suite.waitForRuntimeAndMakeItReady(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)

		// when
		// OSB update:
		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
				"context": {
					"globalaccount_id": "g-account-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"oidc": {
						"clientID": "id-ooo",
						"signingAlgs": ["PS512"],
						"issuerURL": "https://issuer.url.com"
					}
				}
			}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		upgradeOperationID := suite.DecodeOperationID(resp)

		suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
		runtime := suite.GetRuntimeResourceByInstanceID(iid)

		assert.Equal(t, "id-ooo", *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID)
		jwks, _ := base64.StdEncoding.DecodeString("andrcy10b2tlbi1kZWZhdWx0")
		assert.Equal(t, jwks, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].JWKS)
		assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 1)
	})
}

func TestUpdateGlobalAccountID(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.Broker.SubaccountMovementEnabled = true
	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
	opID := suite.DecodeOperationID(resp)
	suite.waitForRuntimeAndMakeItReady(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
				   "context": {
					   "sm_operator_credentials": {
						   "clientid": "cid",
						   "clientsecret": "cs",
						   "url": "url",
						   "sm_url": "sm_url"
					   },
					   "globalaccount_id": "new-g-account-id",
					   "subaccount_id": "sub-id",
					   "user_id": "john.smith@email.com"
				   },
					"parameters": {
			}
   }`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	actions, err := suite.db.Actions().ListActionsByInstanceID(iid)
	assert.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, actions[0].Type, pkg.SubaccountMovementActionType)
	assert.Equal(t, actions[0].Message, "Subaccount sub-id moved from Global Account g-account-id to new-g-account-id.")
	assert.Equal(t, actions[0].OldValue, "g-account-id")
	assert.Equal(t, actions[0].NewValue, "new-g-account-id")
}

func TestUpdate_ZonesDiscovery(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.ProvidersConfigurationFilePath = providersZonesDiscovery

	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"region": "us-east-1",
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								}
							]
						}
   			}`)
	opID := suite.DecodeOperationID(resp)
	suite.waitForRuntimeAndMakeItReady(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 1)
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-1", 3, "zone-d", "zone-e", "zone-f", "zone-g")

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-2",
									"machineType": "m5.xlarge",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-3",
									"machineType": "c7i.large",
									"haZones": false,
									"autoScalerMin": 1,
									"autoScalerMax": 1
								}
							]
						}
   			}`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime = suite.GetRuntimeResourceByInstanceID(iid)
	assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 3)
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-1", 3, "zone-d", "zone-e", "zone-f", "zone-g")
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-2", 3, "zone-h", "zone-i", "zone-j", "zone-k")
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-3", 1, "zone-l", "zone-m")
}

func TestUpdateClusterName(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
	opID := suite.DecodeOperationID(resp)
	suite.waitForRuntimeAndMakeItReady(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)

	gotInstance := suite.GetInstance(iid)
	assert.Equal(t, "testing-cluster", gotInstance.Parameters.Parameters.Name)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
						"name": "updated-name"
			}
   }`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	gotInstance = suite.GetInstance(iid)
	assert.Equal(t, "updated-name", gotInstance.Parameters.Parameters.Name)
}

func TestUpdate_CredentialsBinding(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.ProvidersConfigurationFilePath = providersZonesDiscovery
	cfg.SubscriptionGardenerResource = "CredentialsBinding"

	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
							"region": "us-east-1",
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								}
							]
						}
   			}`)
	opID := suite.DecodeOperationID(resp)
	suite.waitForRuntimeAndMakeItReady(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 1)
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-1", 3, "zone-d", "zone-e", "zone-f", "zone-g")

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       					"context": {
           					"globalaccount_id": "g-account-id",
           					"user_id": "john.smith@email.com"
       					},
						"parameters": {
							"additionalWorkerNodePools": [
								{
									"name": "name-1",
									"machineType": "m6i.large",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-2",
									"machineType": "m5.xlarge",
									"haZones": true,
									"autoScalerMin": 3,
									"autoScalerMax": 20
								},
								{
									"name": "name-3",
									"machineType": "c7i.large",
									"haZones": false,
									"autoScalerMin": 1,
									"autoScalerMax": 1
								}
							]
						}
   			}`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	runtime = suite.GetRuntimeResourceByInstanceID(iid)
	assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 3)
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-1", 3, "zone-d", "zone-e", "zone-f", "zone-g")
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-2", 3, "zone-h", "zone-i", "zone-j", "zone-k")
	suite.assertAdditionalWorkerZones(t, runtime.Spec.Shoot.Provider, "name-3", 1, "zone-l", "zone-m")
}
