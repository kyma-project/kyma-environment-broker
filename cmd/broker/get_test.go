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

func TestGetParametersAfterProvisioning_InstanceWithCustomOidcConfig(t *testing.T) {
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
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
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
	}`, iid, iid), string(r))
}

func TestGetParametersAfterProvisioning_InstanceWithNoOidcConfig(t *testing.T) {
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
						"region": "eu-central-1"
					}
		}`)
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
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
	}`, iid, iid), string(r))
}

func TestGetParametersAfterProvisioning_InstanceWithListOidcConfig(t *testing.T) {
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
						"oidc": {
							"list": [
								{
									"clientID": "client-id-oidc",
									"groupsClaim": "groups",
									"issuerURL": "https://isssuer.url",
									"signingAlgs": ["RS256"],
									"usernameClaim": "sub",
									"groupsPrefix": "-",
									"usernamePrefix": "-"
								}
							]
						}
					}
		}`)
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
				"name": "testing-cluster",
				"oidc": {
					"list": [
						{
							"clientID": "client-id-oidc",
							"groupsClaim": "groups",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": ["RS256"],
							"usernameClaim": "sub",
							"groupsPrefix": "-",
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
	}`, iid, iid), string(r))
}

func TestGetParametersAfterProvisioning_InstanceWithEmptyListOidcConfig(t *testing.T) {
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
						"oidc": {
							"list": []
						}
					}
		}`)
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
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
			},
			"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
			"platform_provider": "Azure",
			"platform_region": "cf-eu21",
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
		},
		"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
		"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	}`, iid, iid), string(r))
}

func TestGetParametersAfterProvisioning_InstanceWithCustomOidcConfigWithGroupsPrefixAndRequiredClaimsThatShouldBeIgnored(t *testing.T) {
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
						"oidc": {				
							"clientID": "client-id-oidc",
							"groupsClaim": "groups",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": [
									"RS256"
							],
							"usernameClaim": "sub",
							"usernamePrefix": "-",
							"groupsPrefix": "abcd",
							"requiredClaims": ["claim=value"]
						}
					}
		}`)
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
				"name": "testing-cluster",
				"oidc": {
					"clientID": "client-id-oidc",
					"groupsClaim": "groups",
					"issuerURL": "https://isssuer.url",
					"signingAlgs": ["RS256"],
					"usernameClaim": "sub",
					"usernamePrefix": "-",
                    "requiredClaims": ["claim=value"],
                    "groupsPrefix": "abcd"
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
	}`, iid, iid), string(r))
}

func TestGetParametersAfterProvisioning_InstanceWithCustomOidcConfigWithGroupsPrefixAndRequiredClaimsThatShouldBeReturned(t *testing.T) {
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
						"oidc": {				
							"clientID": "client-id-oidc",
							"groupsClaim": "groups",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": [
									"RS256"
							],
							"usernameClaim": "sub",
							"usernamePrefix": "-",
							"groupsPrefix": "-",
							"requiredClaims": ["claim=value"]
						}
					}
		}`)
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com"
			},
			"parameters": {
				"name": "testing-cluster",
				"oidc": {
					"clientID": "client-id-oidc",
					"groupsClaim": "groups",
					"issuerURL": "https://isssuer.url",
					"signingAlgs": ["RS256"],
					"usernameClaim": "sub",
					"usernamePrefix": "-",
					"groupsPrefix": "-",
					"requiredClaims": ["claim=value"]
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
	}`, iid, iid), string(r))
}

func TestGetParametersAfterUpdate_InstanceWithObjectOidcUpdatedWithObjectOidc(t *testing.T) {
	// given
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
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
	defer func() { _ = resp.Body.Close() }()
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com",
			"active": true
			},
			"parameters": {
				"name": "testing-cluster",
				"oidc": {
					"clientID": "id-ooo",
					"groupsClaim": "",
					"issuerURL": "https://issuer.url.com",
					"signingAlgs": ["RS256"],
					"usernameClaim": "",
					"usernamePrefix": ""
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
	}`, iid, iid), string(r))
}

func TestGetParametersAfterUpdate_InstanceWithObjectOidcUpdatedWithListOidc(t *testing.T) {
	// given
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
						"oidc": {				
							"clientID": "client-id-oidc",
							"groupsClaim": "groups",
							"groupsPrefix": "-",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": [
									"RS256"
							],
							"usernameClaim": "sub",
							"usernamePrefix": "-"
						}
					}
		}`)
	defer func() { _ = resp.Body.Close() }()

	opID := suite.DecodeOperationID(resp)
	suite.processKIMProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
					"list": [
						{
							"clientID": "client-id-oidc2",
							"groupsClaim": "groups",
							"groupsPrefix": "-",
							"issuerURL": "https://isssuer.url",
							"signingAlgs": ["RS256"],
							"usernameClaim": "sub",
							"usernamePrefix": "-"
						}
					]
				}
		}
   }`)
	defer func() { _ = resp.Body.Close() }()
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	// then
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	defer func() { _ = resp.Body.Close() }()
	r, e := io.ReadAll(resp.Body)
	require.NoError(t, e)
	assert.JSONEq(t, fmt.Sprintf(`{
		"dashboard_url": "/?kubeconfigID=%s",
		"metadata": {
			"labels": {
				"Name": "testing-cluster",
				"APIServerURL": "https://api.server.url.dummy",
				"KubeconfigURL": "https:///kubeconfig/%s"
			}
		},
		"parameters": {
			"ers_context": {
			"globalaccount_id": "g-account-id",
			"subaccount_id": "sub-id",
			"user_id": "john.smith@email.com",
		    "active": true
			},
			"parameters": {
				"name": "testing-cluster",
				"oidc": {
					"list": [
						{
							"clientID": "client-id-oidc2",
							"groupsClaim": "groups",
							"groupsPrefix": "-",
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
	}`, iid, iid), string(r))
}
