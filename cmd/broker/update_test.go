package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const updateRequestPathFormat = "oauth/v2/service_instances/%s?accepts_incomplete=true"

func TestUpdate(t *testing.T) {
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
							"clientID": "id-initial",
							"signingAlgs": ["PS512"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})
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
	suite.failProvisioningByOperationID(opID)

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
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})
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
	suite.processProvisioningByOperationID(opID)

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
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig:                    defaultOIDCConfig(),
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})

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
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})
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
	suite.processProvisioningByOperationID(opID)
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
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
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
	suite.processProvisioningByInstanceID(iid)

	suite.AssertKymaResourceNotExists(opID)
	instance := suite.GetInstance(iid)
	assert.Equal(t, instance.RuntimeID, instance.InstanceDetails.KymaResourceName)
	assert.Equal(t, instance.RuntimeID, instance.InstanceDetails.GardenerClusterName)
	suite.AssertKymaResourceExistsByInstanceID(iid)
}

func TestUnsuspensionTrialKyma20(t *testing.T) {
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
	suite.processProvisioningByOperationID(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.fixServiceBindingAndInstances(t)

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

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	suite.assertServiceBindingAndInstancesAreRemoved(t)

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
	suite.processProvisioningByInstanceID(iid)

	suite.AssertKymaResourceNotExists(opID)
	suite.AssertKymaLabelNotExists(opID, "kyma-project.io/platform-region")
	suite.AssertKymaResourceExistsByInstanceID(iid)
}

func TestUnsuspensionTrialWithDefaultProviderChangedForNonDefaultRegion(t *testing.T) {
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-us10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.Log("*** Suspension ***")

	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-us10/v2/service_instances/%s?accepts_incomplete=true", iid),
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

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// WHEN
	suite.ChangeDefaultTrialProvider(pkg.AWS)
	// OSB update
	suite.Log("*** Unsuspension ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-us10/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	suite.processProvisioningByInstanceID(iid)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// check that the region and zone is set
	suite.AssertAWSRegionAndZone("us-east-1")

	suite.AssertKymaResourceNotExists(opID)
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
	// uncomment to see graphql queries
	//suite.EnableDumpingProvisionerRequests()
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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
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
	suite.processProvisioningByInstanceID(iid)

	// THEN
	suite.WaitForLastOperation(iid, domain.Succeeded)
	instance = suite.GetInstance(iid)
	assert.Equal(t, "id-oooxx", instance.Parameters.Parameters.OIDC.ClientID)
	input := suite.LastProvisionInput(iid)
	assert.Equal(t, "id-oooxx", input.ClusterConfig.GardenerConfig.OidcConfig.ClientID)

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
	suite.processProvisioningByOperationID(opID)
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
	expectedAdmins := []string{"john.smith@email.com"}

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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
}

func TestUpdateDefaultAdminNotChangedWithCustomOIDC(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"john.smith@email.com"}

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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
}

func TestUpdateDefaultAdminNotChangedWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"john.smith@email.com"}

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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "new-groups-claim",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS384"},
				UsernameClaim:  "new-username-claim",
				UsernamePrefix: "->",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://newissuer.url.com",
				SigningAlgs:    []string{"ES256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
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
	suite.processProvisioningByOperationID(opID)
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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
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
	suite.processProvisioningByOperationID(opID)
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
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "new-groups-claim",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"ES384"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins,
	})
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
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins1,
	})
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
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	disabled = false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://newissuer.url.com",
				SigningAlgs:    []string{"PS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "->",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: expectedAdmins2,
	})
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
	suite.processProvisioningByOperationID(opID)

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
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	min, max, surge, unav := 15, 25, 10, 7
	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			AutoScalerMin:                 &min,
			AutoScalerMax:                 &max,
			MaxSurge:                      &surge,
			MaxUnavailable:                &unav,
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})
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
	suite.processProvisioningByOperationID(opID)

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
	suite.processProvisioningByOperationID(opID)

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

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	max := 15
	disabled := false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			AutoScalerMax:                 &max,
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})

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
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)
	min := 14
	disabled = false
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			AutoScalerMin:                 &min,
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})

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
	// uncomment to see graphql queries
	//suite.EnableDumpingProvisionerRequests()
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
	suite.processProvisioningByOperationID(opID)
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
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

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
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	i, err := suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "instance after provisioning")
	rs, err := suite.db.RuntimeStates().ListByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "runtime states after provisioning")
	assert.Equal(t, 1, len(rs), "runtime states after provisioning")
	assert.Equal(t, "m5.xlarge", rs[0].ClusterConfig.MachineType, "after provisioning")

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

	//c: [ccv1]

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	// check call to provisioner that machine type has been updated
	rs, err = suite.db.RuntimeStates().ListByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "runtime states after update")
	assert.Equal(t, 2, len(rs), "runtime states after update")
	assert.Equal(t, "m5.2xlarge", rs[0].ClusterConfig.MachineType, "after update")
}

func TestUpdateBTPOperatorCredsSuccess(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := "InstanceID-BTPOperator"

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
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	i, err := suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after provisioning, before update")
	rs, err := suite.db.RuntimeStates().GetLatestByRuntimeID(i.RuntimeID)
	assert.Equal(t, opID, rs.OperationID, "runtime state provisioning operation ID")
	assert.NoError(t, err, "getting runtime state after provisioning, before update")
	assert.ElementsMatch(t, rs.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})

	rsu1, err := suite.db.RuntimeStates().GetLatestByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after update")
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after update")
	assert.ElementsMatch(t, rsu1.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
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
	}
}`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	rsu2, err := suite.db.RuntimeStates().GetLatestByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after update")
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after update")
	assert.ElementsMatch(t, rsu2.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})
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
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then
	disabled := true
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
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
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	updateOp, _ := suite.db.Operations().GetOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, updateOperationID, &disabled)
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
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then
	disabled := false
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
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

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	updateOp, _ := suite.db.Operations().GetOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	instance2 := suite.GetInstance(id)
	// license_type should be stored in the instance table for ERS context and future upgrades
	disabled = true
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, updateOperationID, &disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
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
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	instance := suite.GetInstance(id)

	// then
	disabled := false
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, opID, &disabled)
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
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
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
	suite.WaitForLastOperation(id, domain.InProgress)
	suite.FinishUpdatingOperationByProvisioner(updateOperation2ID)
	suite.WaitForOperationState(updateOperation2ID, domain.Succeeded)
	instance3 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance3.Parameters.ErsContext.LicenseType)
	disabled = true
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, updateOperation2ID, &disabled)
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
	suite.processProvisioningByOperationID(opID)
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
	suite.processProvisioningByOperationID(opID)
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
	suite.processProvisioningByOperationID(opID)
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
