// build provisioning-test
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

const (
	workersAmount                 int = 5
	provisioningRequestPathFormat     = "oauth/cf-eu10/v2/service_instances/%s"
)

func TestCatalog(t *testing.T) {
	// this test is used for human-testing the catalog response
	t.Skip()
	catalogTestFile := "catalog-test.json"
	catalogTestFilePerm := os.FileMode.Perm(0666)
	outputToFile := false
	prettyJson := false
	prettify := func(content []byte) *bytes.Buffer {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, content, "", "    ")
		assert.NoError(t, err)
		return &prettyJSON
	}

	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()

	// when
	resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/catalog"), ``)

	content, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	defer resp.Body.Close()

	if outputToFile {
		if prettyJson {
			err = os.WriteFile(catalogTestFile, prettify(content).Bytes(), catalogTestFilePerm)
			assert.NoError(t, err)
		} else {
			err = os.WriteFile(catalogTestFile, content, catalogTestFilePerm)
			assert.NoError(t, err)
		}
	} else {
		if prettyJson {
			fmt.Println(prettify(content).String())
		} else {
			fmt.Println(string(content))
		}
	}
}

func TestProvisioning_HappyPath(t *testing.T) {
	// given
	suite := NewProvisioningSuite(t, false, "", false)
	defer suite.TearDown()

	// when
	provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{})

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

	// when
	suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
	suite.AssertAllStagesFinished(provisioningOperationID)
	suite.AssertProvisioningRequest()
}

func TestProvisioningWithKIM(t *testing.T) {
	// this test is used for developing KIM integration. The main reason is to just print the Runtime CR as YAML as a result of a process

	cfg := fixConfig()
	cfg.Broker.KimConfig.Enabled = true
	cfg.Broker.KimConfig.Plans = []string{"aws", "preview"}
	cfg.Broker.KimConfig.DryRun = true

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
						"autoScalerMax": 11,
                        "autoScalerMin": 6
					}
		}`)

	opID := suite.DecodeOperationID(resp)

	suite.processProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)
}

func TestProvisioningWithKIMOnlyForTrial(t *testing.T) {

	cfg := fixConfig()
	cfg.Broker.KimConfig.Enabled = true
	cfg.Broker.KimConfig.Plans = []string{"trial"}
	cfg.Broker.KimConfig.KimOnlyPlans = []string{"trial"}

	cfg.Provisioner.DefaultTrialProvider = pkg.AWS

	suite := NewBrokerSuiteTestWithConfig(t, cfg)
	defer suite.TearDown()
	iid := uuid.New().String()
	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu21/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
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

	suite.processKIMOnlyProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)
	suite.AssertRuntimeResourceLabels(opID)
}

func TestProvisioningWithKIMOnlyForAWS(t *testing.T) {

	cfg := fixConfig()
	cfg.Broker.KimConfig.Enabled = true
	cfg.Broker.KimConfig.Plans = []string{"aws"}
	cfg.Broker.KimConfig.KimOnlyPlans = []string{"aws"}

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

	suite.processKIMOnlyProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)
	suite.AssertRuntimeResourceLabels(opID)
}

func TestProvisioning_HappyPathAWS(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaAnnotationExists(opID, "compass-runtime-id-for-migration")
	suite.AssertKymaLabelsExist(opID, map[string]string{"kyma-project.io/region": "eu-central-1", "kyma-project.io/provider": "AWS"})
	suite.AssertKymaLabelNotExists(opID, "kyma-project.io/platform-region")
}

func TestProvisioning_SeedAndShootSameRegion_Disabled(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
						"region": "eu-central-1",
						"shootAndSeedSameRegion": true
					}
		}`)
	opID := suite.DecodeOperationID(resp)

	suite.processProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)
	assert.False(t, suite.provisionerClient.IsSeedAndRegionValidationEnabled())
}

func TestProvisioning_SeedAndShootSameRegion_Enabled(t *testing.T) {
	// given
	cfg := fixConfig()
	cfg.Broker.EnableShootAndSeedSameRegion = true
	cfg.Provisioner.EnableShootAndSeedSameRegion = true
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
						"region": "eu-central-1",
						"shootAndSeedSameRegion": true
					}
		}`)
	opID := suite.DecodeOperationID(resp)

	suite.processProvisioningByOperationID(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)
	assert.True(t, suite.provisionerClient.IsSeedAndRegionValidationEnabled())
}

func TestProvisioning_HappyPathSapConvergedCloud(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	t.Run("should provision SAP Converged Cloud", func(t *testing.T) {
		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu20-staging/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
						"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
						"plan_id": "03b812ac-c991-4528-b5bd-08b303523a63",
						"context": {
							"globalaccount_id": "g-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
						},
						"parameters": {
							"name": "testing-cluster",
							"region": "eu-de-1"
						}
			}`)
		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		// then
		suite.WaitForOperationState(opID, domain.Succeeded)

		suite.AssertKymaResourceExists(opID)
		suite.AssertKymaAnnotationExists(opID, "compass-runtime-id-for-migration")
		suite.AssertKymaLabelsExist(opID, map[string]string{"kyma-project.io/region": "eu-de-1", "kyma-project.io/provider": "SapConvergedCloud"})
	})

	t.Run("should fail for invalid platform region - invalid platform region", func(t *testing.T) {
		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/invalid/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "03b812ac-c991-4528-b5bd-08b303523a63",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "eu-de-1"
					}
		}`)
		parsedResponse := suite.ReadResponse(resp)
		assert.Contains(t, string(parsedResponse), "plan-id not in the catalog")
	})

	t.Run("should fail for invalid platform region - default platform region", func(t *testing.T) {

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "03b812ac-c991-4528-b5bd-08b303523a63",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "eu-de-1"
					}
		}`)
		parsedResponse := suite.ReadResponse(resp)
		assert.Contains(t, string(parsedResponse), "plan-id not in the catalog")
	})

	t.Run("should fail for invalid platform region - invalid Kyma region", func(t *testing.T) {

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu20-staging/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "03b812ac-c991-4528-b5bd-08b303523a63",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "invalid"
					}
		}`)
		parsedResponse := suite.ReadResponse(resp)
		assert.Contains(t, string(parsedResponse), "while validating input parameters: region: region must be one of the following")
	})

}

func TestProvisioning_Preview(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region": "eu-central-1"
					}
		}`)
	opID := suite.DecodeOperationID(resp)

	suite.waitForRuntimeAndMakeItReady(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.AssertKymaResourceExists(opID)
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":   "eu-central-1",
		"kyma-project.io/provider": "AWS",
	})
	suite.AssertKymaLabelNotExists(opID, "kyma-project.io/platform-region")
}

func TestProvisioning_NetworkingParametersForAWS(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
		
				"context": {
					"globalaccount_id": "e449f875-b5b2-4485-b7c0-98725c0571bf",
						"subaccount_id": "test",
					"user_id": "piotr.miskiewicz@sap.com"
					
				},
				"parameters": {
					"name": "test",
					"region": "eu-central-1",
					"networking": {
						"nodes": "192.168.48.0/20"
					}
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)

	suite.processProvisioningByOperationID(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)
}

func TestProvisioning_AllNetworkingParametersForAWS(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
				"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
		
				"context": {
					"globalaccount_id": "e449f875-b5b2-4485-b7c0-98725c0571bf",
						"subaccount_id": "test",
					"user_id": "piotr.miskiewicz@sap.com"
					
				},
				"parameters": {
					"name": "test",
					"region": "eu-central-1",
					"networking": {
						"nodes": "192.168.48.0/20",
						"pods": "10.104.0.0/24",
						"services": "10.105.0.0/24"
					}
				}
			}
		}`)
	opID := suite.DecodeOperationID(resp)

	suite.processProvisioningByOperationID(opID)

	suite.WaitForOperationState(opID, domain.Succeeded)
}

func TestProvisioning_AWSWithEURestrictedAccessBadRequest(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "not-whitelisted-global-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region":"us-west-1"
					}
		}`)
	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProvisioning_AzureWithEURestrictedAccessBadRequest(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "not-whitelisted-global-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region":"japaneast"
					}
		}`)
	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProvisioning_AzureWithEURestrictedAccessHappyFlow(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "whitelisted-global-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region":"switzerlandnorth"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	suite.AssertAzureRegion("switzerlandnorth")
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "switzerlandnorth",
		"kyma-project.io/provider":        "Azure",
		"kyma-project.io/platform-region": "cf-ch20"})
}

func TestProvisioning_AzureWithEURestrictedAccessDefaultRegion(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "whitelisted-global-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "switzerlandnorth"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	suite.AssertAzureRegion("switzerlandnorth")
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "switzerlandnorth",
		"kyma-project.io/provider":        "Azure",
		"kyma-project.io/platform-region": "cf-ch20"})
}

func TestProvisioning_AWSWithEURestrictedAccessHappyFlow(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "whitelisted-global-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region":"eu-central-1"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	suite.AssertAWSRegionAndZone("eu-central-1")
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-central-1",
		"kyma-project.io/provider":        "AWS",
		"kyma-project.io/platform-region": "cf-eu11"})

}

func TestProvisioning_AWSWithEURestrictedAccessDefaultRegion(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "whitelisted-global-account-id",
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
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	suite.AssertAWSRegionAndZone("eu-central-1")
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-central-1",
		"kyma-project.io/provider":        "AWS",
		"kyma-project.io/platform-region": "cf-eu11"})

}

func TestProvisioning_TrialWithEmptyRegion(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region":""
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// then
	suite.AssertAWSRegionAndZone("eu-west-1")
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/provider": "AWS",
		"kyma-project.io/region":   "eu-west-1"})
	suite.AssertKymaLabelNotExists(opID, "kyma-project.io/platform-region")

}

func TestProvisioning_Conflict(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
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
	resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster-2"
					}
		}`)
	// then
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestProvisioning_OwnCluster(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"kubeconfig":"YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmN1cnJlbnQtY29udGV4dDogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKY29udGV4dHM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY29udGV4dDoKICAgICAgY2x1c3Rlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKICAgICAgdXNlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUtdG9rZW4KY2x1c3RlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2FwaS5jbHVzdGVyLW5hbWUua3ltYS1kZXYuc2hvb3QuY2FuYXJ5Lms4cy1oYW5hLm9uZGVtYW5kLmNvbQogICAgICBjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YTogPi0KICAgICAgICBMUzB0TFMxQ1JVZEpUaUJEUlZKVVNVWkpRMEZVUlMwdExTMHQKdXNlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZS10b2tlbgogICAgdXNlcjoKICAgICAgdG9rZW46ID4tCiAgICAgICAgdE9rRW4K",
						"shootName": "sh1",
						"shootDomain": "sh1.avs.sap.nothing"
					}
		}`)
	opID := suite.DecodeOperationID(resp)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	// get instance OSB API call
	// when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	r, e := io.ReadAll(resp.Body)

	// then
	require.NoError(t, e)
	assert.JSONEq(t, `{
  "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
  "parameters": {
    "plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
    "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
    "ers_context": {
      "subaccount_id": "sub-id",
      "globalaccount_id": "g-account-id",
      "user_id": "john.smith@email.com"
    },
    "parameters": {
      "name": "testing-cluster",
      "shootName": "sh1",
      "shootDomain": "sh1.avs.sap.nothing"
    },
    "platform_region": "",
    "platform_provider": "unknown"
  },
  "metadata": {
    "labels": {
      "Name": "testing-cluster"
    }
  }
}`, string(r))

}

func TestProvisioning_TrialAtEU(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "whitelisted-global-account-id",
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

	// then
	suite.AssertAWSRegionAndZone("eu-central-1")
	suite.AssertKymaLabelsExist(opID, map[string]string{
		"kyma-project.io/region":          "eu-central-1",
		"kyma-project.io/provider":        "AWS",
		"kyma-project.io/platform-region": "cf-eu11",
	})

}

func TestProvisioning_HandleExistingOperation(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	firstResp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)

	secondResp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    },
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)

	firstBodyBytes, _ := io.ReadAll(firstResp.Body)
	secondBodyBytes, _ := io.ReadAll(secondResp.Body)

	// then
	assert.Equal(t, string(firstBodyBytes), string(secondBodyBytes))
}

func TestProvisioning_ClusterParameters(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID                       string
		platformRegion               string
		platformProvider             pkg.CloudProvider
		region                       string
		multiZone                    bool
		controlPlaneFailureTolerance string
		useSmallerMachineTypes       bool

		expectedZonesCount                  *int
		expectedProvider                    string
		expectedMinimalNumberOfNodes        int
		expectedMaximumNumberOfNodes        int
		expectedMachineType                 string
		expectedSharedSubscription          bool
		expectedSubscriptionHyperscalerType hyperscaler.Type
	}{
		"Regular trial": {
			planID: broker.TrialPlanID,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedMachineType:                 "Standard_D4s_v5",
			expectedProvider:                    "azure",
			expectedSharedSubscription:          true,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure(),
		},
		"Regular trial with smaller machines": {
			planID:                 broker.TrialPlanID,
			useSmallerMachineTypes: true,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedMachineType:                 "Standard_D2s_v5",
			expectedProvider:                    "azure",
			expectedSharedSubscription:          true,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure(),
		},
		"Freemium aws": {
			planID:           broker.FreemiumPlanID,
			platformProvider: pkg.AWS,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedMachineType:                 "m5.xlarge",
			expectedSubscriptionHyperscalerType: hyperscaler.AWS(),
		},
		"Freemium aws with smaller machines": {
			planID:                 broker.FreemiumPlanID,
			platformProvider:       pkg.AWS,
			useSmallerMachineTypes: true,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedMachineType:                 "m6i.large",
			expectedSubscriptionHyperscalerType: hyperscaler.AWS(),
		},
		"Freemium azure": {
			planID:           broker.FreemiumPlanID,
			platformProvider: pkg.Azure,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedMachineType:                 "Standard_D4s_v5",
			expectedSubscriptionHyperscalerType: hyperscaler.Azure(),
		},
		"Freemium azure with smaller machines": {
			planID:                 broker.FreemiumPlanID,
			platformProvider:       pkg.Azure,
			useSmallerMachineTypes: true,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedMachineType:                 "Standard_D2s_v5",
			expectedSubscriptionHyperscalerType: hyperscaler.Azure(),
		},
		"Production Azure": {
			planID:    broker.AzurePlanID,
			region:    "westeurope",
			multiZone: false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultAzureMachineType,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure(),
		},
		"Production Multi-AZ Azure": {
			planID:                       broker.AzurePlanID,
			region:                       "westeurope",
			multiZone:                    true,
			controlPlaneFailureTolerance: "zone",

			expectedZonesCount:                  ptr.Integer(3),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultAzureMachineType,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure(),
		},
		"Production AWS": {
			planID:    broker.AWSPlanID,
			region:    "us-east-1",
			multiZone: false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultAWSMachineType,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.AWS(),
		},
		"Production Multi-AZ AWS": {
			planID:                       broker.AWSPlanID,
			region:                       "us-east-1",
			multiZone:                    true,
			controlPlaneFailureTolerance: "zone",

			expectedZonesCount:                  ptr.Integer(3),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultAWSMachineType,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.AWS(),
		},
		"Production GCP": {
			planID:    broker.GCPPlanID,
			region:    "us-central1",
			multiZone: false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultGCPMachineType,
			expectedProvider:                    "gcp",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.GCP("cf-us30"),
		},
		"Production GCP KSA": {
			planID:         broker.GCPPlanID,
			platformRegion: "cf-sa30",
			region:         "me-central2",
			multiZone:      false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultGCPMachineType,
			expectedProvider:                    "gcp",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.GCP("cf-sa30"),
		},
		"Production Multi-AZ GCP": {
			planID:                       broker.GCPPlanID,
			region:                       "us-central1",
			multiZone:                    true,
			controlPlaneFailureTolerance: "zone",

			expectedZonesCount:                  ptr.Integer(3),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 provider.DefaultGCPMachineType,
			expectedProvider:                    "gcp",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.GCP("cf-us30"),
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			suite := NewProvisioningSuite(t, tc.multiZone, tc.controlPlaneFailureTolerance, tc.useSmallerMachineTypes)
			defer suite.TearDown()

			// when
			provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{
				PlanID:           tc.planID,
				PlatformRegion:   tc.platformRegion,
				PlatformProvider: tc.platformProvider,
				Region:           tc.region,
			})

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
			suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

			// when
			suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
			suite.AssertAllStagesFinished(provisioningOperationID)

			suite.AssertProvider(tc.expectedProvider)
			suite.AssertMinimalNumberOfNodes(tc.expectedMinimalNumberOfNodes)
			suite.AssertMaximumNumberOfNodes(tc.expectedMaximumNumberOfNodes)
			suite.AssertMachineType(tc.expectedMachineType)
			suite.AssertZonesCount(tc.expectedZonesCount, tc.planID)
			suite.AssertSubscription(tc.expectedSharedSubscription, tc.expectedSubscriptionHyperscalerType)
			suite.AssertControlPlaneFailureTolerance(tc.controlPlaneFailureTolerance)
		})

	}
}

func TestProvisioning_OIDCValues(t *testing.T) {

	t.Run("should apply default OIDC values when OIDC object is nil", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		defaultOIDC := fixture.FixOIDCConfigDTO()
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       defaultOIDC.ClientID,
			GroupsClaim:    defaultOIDC.GroupsClaim,
			IssuerURL:      defaultOIDC.IssuerURL,
			SigningAlgs:    defaultOIDC.SigningAlgs,
			UsernameClaim:  defaultOIDC.UsernameClaim,
			UsernamePrefix: defaultOIDC.UsernamePrefix,
		}

		// when
		provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{})

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply default OIDC values when all OIDC object's fields are empty", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		defaultOIDC := fixture.FixOIDCConfigDTO()
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       defaultOIDC.ClientID,
			GroupsClaim:    defaultOIDC.GroupsClaim,
			IssuerURL:      defaultOIDC.IssuerURL,
			SigningAlgs:    defaultOIDC.SigningAlgs,
			UsernameClaim:  defaultOIDC.UsernameClaim,
			UsernamePrefix: defaultOIDC.UsernamePrefix,
		}
		options := RuntimeOptions{
			OIDC: &pkg.OIDCConfigDTO{},
		}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply provided OIDC configuration", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		providedOIDC := pkg.OIDCConfigDTO{
			ClientID:       "fake-client-id-1",
			GroupsClaim:    "fakeGroups",
			IssuerURL:      "https://testurl.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "fakeUsernameClaim",
			UsernamePrefix: "::",
		}
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       providedOIDC.ClientID,
			GroupsClaim:    providedOIDC.GroupsClaim,
			IssuerURL:      providedOIDC.IssuerURL,
			SigningAlgs:    providedOIDC.SigningAlgs,
			UsernameClaim:  providedOIDC.UsernameClaim,
			UsernamePrefix: providedOIDC.UsernamePrefix,
		}
		options := RuntimeOptions{OIDC: &providedOIDC}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply default OIDC values on empty OIDC params from input", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		providedOIDC := pkg.OIDCConfigDTO{
			ClientID:  "fake-client-id-1",
			IssuerURL: "https://testurl.local",
		}
		defaultOIDC := defaultOIDCValues()
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       providedOIDC.ClientID,
			GroupsClaim:    defaultOIDC.GroupsClaim,
			IssuerURL:      providedOIDC.IssuerURL,
			SigningAlgs:    defaultOIDC.SigningAlgs,
			UsernameClaim:  defaultOIDC.UsernameClaim,
			UsernamePrefix: defaultOIDC.UsernamePrefix,
		}
		options := RuntimeOptions{OIDC: &providedOIDC}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})
}

func TestProvisioning_RuntimeAdministrators(t *testing.T) {
	t.Run("should use UserID as default value for admins list", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		options := RuntimeOptions{
			UserID: "fake-user-id",
		}
		expectedAdmins := []string{"fake-user-id"}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})

	t.Run("should apply new admins list", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		options := RuntimeOptions{
			UserID:        "fake-user-id",
			RuntimeAdmins: []string{"admin1@test.com", "admin2@test.com"},
		}
		expectedAdmins := []string{"admin1@test.com", "admin2@test.com"}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})

	t.Run("should apply empty admin value (list is not empty)", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "", false)
		defer suite.TearDown()
		options := RuntimeOptions{
			UserID:        "fake-user-id",
			RuntimeAdmins: []string{""},
		}
		expectedAdmins := []string{""}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})
}

func TestProvisioning_WithoutNetworkFilter(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
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
	instance := suite.GetInstance(iid)

	// then
	disabled := false
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
	assert.Nil(suite.t, instance.Parameters.ErsContext.LicenseType)
}

func TestProvisioning_WithNetworkFilter(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
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
	instance := suite.GetInstance(iid)

	// then

	disabled := true
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance.Parameters.ErsContext.LicenseType)
}

func TestProvisioning_Modules(t *testing.T) {

	const defaultModules = "kyma-with-keda-and-btp-operator.yaml"

	t.Run("with given custom list of modules [btp-operator, ked] all set", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "test",
						"region": "eu-central-1",
						"modules": {
							"list": [
								{
									"name": "btp-operator",
									"customResourcePolicy": "Ignore",
									"channel": "fast"
								},
								{
									"name": "keda",
									"customResourcePolicy": "CreateAndDelete",
									"channel": "regular"
								}
							]
						}
					}
				}`)
		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)
		op, err := suite.db.Operations().GetOperationByID(opID)
		assert.NoError(t, err)
		assert.YAMLEq(t, internal.GetKymaTemplateForTests(t, "kyma-with-keda-and-btp-operator-all-params-set.yaml"), op.KymaTemplate)
	})

	t.Run("with given custom list of modules [btp-operator, ked] with channel and crPolicy as empty", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "test",
						"region": "eu-central-1",
						"modules": {
							"list": [
								{
									"name": "btp-operator"
								},
								{
									"name": "keda"
								}
							]
						}
					}
				}`)
		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)
		op, err := suite.db.Operations().GetOperationByID(opID)
		assert.NoError(t, err)
		assert.YAMLEq(t, internal.GetKymaTemplateForTests(t, "kyma-with-keda-and-btp-operator-only-name.yaml"), op.KymaTemplate)
	})

	t.Run("with given custom list of modules [btp-operator, ked] with channel and crPolicy as not set", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "test",
						"region": "eu-central-1",
						"modules": {
							"list": [
								{
									"name": "btp-operator",
									"customResourcePolicy": "",
									"channel": ""
								},
								{
									"name": "keda",
									"customResourcePolicy": "",
									"channel": ""
								}
							]
						}
					}
				}`)
		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)
		op, err := suite.db.Operations().GetOperationByID(opID)
		assert.NoError(t, err)
		assert.YAMLEq(t, internal.GetKymaTemplateForTests(t, "kyma-with-keda-and-btp-operator-only-name.yaml"), op.KymaTemplate)
	})

	t.Run("with given empty list of modules", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "test",
						"region": "eu-central-1",
						"modules": {
							"list": []
						}
					}
				}`)
		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)
		op, err := suite.db.Operations().GetOperationByID(opID)
		assert.NoError(t, err)
		assert.YAMLEq(t, internal.GetKymaTemplateForTests(t, "kyma-no-modules.yaml"), op.KymaTemplate)
	})

	t.Run("with given default as false", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "test",
						"region": "eu-central-1",
						"modules": {
							"default": false
						}
					}
				}`)

		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)
		op, err := suite.db.Operations().GetOperationByID(opID)
		assert.NoError(t, err)
		assert.YAMLEq(t, internal.GetKymaTemplateForTests(t, "kyma-no-modules.yaml"), op.KymaTemplate)
	})

	t.Run("with given default as true", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "test",
						"modules": {
							"default": true
						}
					}
				}`)

		opID := suite.DecodeOperationID(resp)

		suite.processProvisioningByOperationID(opID)

		suite.WaitForOperationState(opID, domain.Succeeded)
		op, err := suite.db.Operations().GetOperationByID(opID)
		assert.NoError(t, err)
		assert.YAMLEq(t, internal.GetKymaTemplateForTests(t, defaultModules), op.KymaTemplate)
	})

	t.Run("oneOf validation fail when two params are set", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
			"context": {
					"globalaccount_id": "whitelisted-global-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
			},
			"parameters": {
				"name": "test",
				"region": "eu-central-1",
				"modules": {
					"default": false,
					"list": [
						{
							"name": "btp-operator",
							"channel": "regular",
							"customResourcePolicy": "CreateAndDelete"
						},
						{
							"name": "keda",
							"channel": "fast",
							"customResourcePolicy": "Ignore"
						}
					]
				}
			}
		}`)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("oneOf validation fail when no any modules param is set", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
							"name": "test",
							"region": "eu-central-1",
							"modules": {}
						}
					}
				}`)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("validation fail due to incorrect channel/crPolicy", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
							"name": "test",
							"region": "eu-central-1",
							"modules": {
								"list": [
									{
										"name": "btp-operator",
										"channel": "regularWrong",
										"customResourcePolicy": "CreateAndDeleteWrong"
									}
								]
							}
						}
					}
				}`)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("validation fail when name not passed", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
							"name": "test",
							"region": "eu-central-1",
							"modules": {
								"list": [
									{
										"channel": "regularWrong",
										"customResourcePolicy": "CreateAndDeleteWrong"
									}
								]
							}
						}
					}
				}`)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("validation fail when name empty", func(t *testing.T) {
		// given
		suite := NewBrokerSuiteTest(t)
		defer suite.TearDown()
		iid := uuid.New().String()

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
							"globalaccount_id": "whitelisted-global-account-id",
							"subaccount_id": "sub-id",
							"user_id": "john.smith@email.com"
					},
					"parameters": {
							"name": "test",
							"region": "eu-central-1",
							"modules": {
								"list": [
									{
										"name": "",
										"channel": "regularWrong",
										"customResourcePolicy": "CreateAndDeleteWrong"
									}
								]
							}
						}
					}
				}`)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
