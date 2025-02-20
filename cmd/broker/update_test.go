package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const updateRequestPathFormat = "oauth/v2/service_instances/%s?accepts_incomplete=true"

func TestUpdate(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.Database.UseLastOperationID = true
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

	runtime := suite.GetRuntimeResourceByInstanceID(iid)
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig

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
	runtime := suite.GetRuntimeResourceByInstanceID(iid)

	assert.Equal(t, "id-ooo", *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
}

func TestUpdateFailedInstance(t *testing.T) {
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
	suite.failRuntimeByKIM(iid)
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

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu20-staging/v2/service_instances/%s?accepts_incomplete=true&plan_id=03b812ac-c991-4528-b5bd-08b303523a63&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu20-staging/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"RS256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-de-1",
		"kyma-project.io/platform-region": "cf-eu20-staging",
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, defaultOIDCValues().ClientID, *oidc.ClientID)
	assert.Equal(t, defaultOIDCValues().SigningAlgs, oidc.SigningAlgs)
	assert.Equal(t, defaultOIDCValues().IssuerURL, *oidc.IssuerURL)
	assert.Equal(t, defaultOIDCValues().GroupsClaim, *oidc.GroupsClaim)
	assert.Equal(t, defaultOIDCValues().UsernameClaim, *oidc.UsernameClaim)
	assert.Equal(t, defaultOIDCValues().UsernamePrefix, *oidc.UsernamePrefix)
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"RS256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
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
	assert.Equal(t, "id-oooxx", *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)

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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"RS256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"RS384"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "new-groups-claim", *oidc.GroupsClaim)
	assert.Equal(t, "new-username-claim", *oidc.UsernameClaim)
	assert.Equal(t, "->", *oidc.UsernamePrefix)
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "client-id-oidc", *oidc.ClientID)
	assert.Equal(t, []string{"RS256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"ES256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://newissuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "client-id-oidc", *oidc.ClientID)
	assert.Equal(t, []string{"RS256"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url", *oidc.IssuerURL)
	assert.Equal(t, "groups", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
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
	oidc := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
	assert.Equal(t, "id-ooo", *oidc.ClientID)
	assert.Equal(t, []string{"ES384"}, oidc.SigningAlgs)
	assert.Equal(t, "https://issuer.url.com", *oidc.IssuerURL)
	assert.Equal(t, "new-groups-claim", *oidc.GroupsClaim)
	assert.Equal(t, "sub", *oidc.UsernameClaim)
	assert.Equal(t, "-", *oidc.UsernamePrefix)
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

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
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

	assert.Equal(t, gardener.OIDCConfig{
		ClientID:       ptr.String("client-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("https://issuer.url"),
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: ptr.String("-"),
	}, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig)

	assert.Equal(t, []string{"john.smith@email.com"}, runtime.Spec.Security.Administrators)
}

func TestUpdateAutoscalerWrongParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
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

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":15
	}
}`)

	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
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
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
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
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
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
func TestUpdateNetworkFilterPersisted(t *testing.T) {
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

	suite.AssertNetworkFilteringDisabled(instance.InstanceID, true)
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
	suite.AssertNetworkFilteringDisabled(instance.InstanceID, true)
	instance2 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
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
	suite.AssertNetworkFilteringDisabled(instance.InstanceID, false)
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
	suite.AssertNetworkFilteringDisabled(instance.InstanceID, true)
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
	suite.AssertNetworkFilteringDisabled(instance.InstanceID, false)
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
			    "name":"$instance",
			    "administrators":["xyz@sap.com", "xyz@gmail.com", "xyz@abc.com"]
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperation2ID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOperation2ID, domain.Succeeded)
	instance3 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance3.Parameters.ErsContext.LicenseType)
	suite.AssertNetworkFilteringDisabled(instance.InstanceID, true)
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
		cfg.Broker.KimConfig.Enabled = true
		cfg.Broker.KimConfig.Plans = []string{"aws"}
		cfg.Broker.KimConfig.KimOnlyPlans = []string{"aws"}

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
		cfg.Broker.KimConfig.Enabled = true
		cfg.Broker.KimConfig.Plans = []string{"aws"}
		cfg.Broker.KimConfig.KimOnlyPlans = []string{"aws"}

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
		cfg.Broker.KimConfig.Enabled = true
		cfg.Broker.KimConfig.Plans = []string{"aws"}
		cfg.Broker.KimConfig.KimOnlyPlans = []string{"aws"}

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
}
