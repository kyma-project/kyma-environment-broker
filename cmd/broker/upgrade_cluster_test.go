package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterUpgrade_UpgradeAfterUpdateWithNetworkPolicy(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := "InstanceID-UpgradeAfterUpdate"

	// provision Kyma 2.0
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
	"context": {
		"sm_operator_credentials": {
			"clientid": "testClientID",
			"clientsecret": "testClientSecret",
			"sm_url": "https://service-manager.kyma.com",
			"url": "https://test.auth.com",
			"xsappname": "testXsappname"
		},
		"globalaccount_id": "g-account-id",
		"subaccount_id": "sub-id",
		"user_id": "john.smith@email.com"
	},
	"parameters": {
		"name": "testing-cluster",
		"region": "eastus"
	}
}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// provide license_type
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"context": {
		"license_type": "CUSTOMER"
	}
}`)

	// finish the update operation
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	i, err := suite.db.Instances().GetByID(id)
	rsu1, err := suite.db.RuntimeStates().GetLatestByRuntimeID(i.RuntimeID)

	// ensure license type is persisted and network filter enabled
	instance2 := suite.GetInstance(id)
	enabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(i.RuntimeID, updateOperationID, &enabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)

	// run upgrade
	orchestrationResp := suite.CallAPI("POST", "upgrade/cluster", `
{
	"strategy": {
		"type": "parallel",
		"schedule": "immediate",
		"parallel": {
			"workers": 1
		}
	},
	"dryRun": false,
	"targets": {
		"include": [
			{
				"subAccount": "sub-id"
			}
		]
	},
	"kubernetes": {
		"kubernetesVersion": "1.25.0"
	}
}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)
	upgradeClusterOperationID, err := suite.DecodeLastUpgradeClusterOperationIDFromOrchestration(oID)
	require.NoError(t, err)

	suite.WaitForOperationState(upgradeClusterOperationID, domain.InProgress)
	suite.FinishUpgradeClusterOperationByProvisioner(upgradeClusterOperationID)
	suite.WaitForOperationState(upgradeClusterOperationID, domain.Succeeded)

	_, err = suite.db.Operations().GetUpgradeClusterOperationByID(upgradeClusterOperationID)
	require.NoError(t, err)

	// ensure component list after upgrade didn't get changed
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after upgrade")
	rsu2, err := suite.db.RuntimeStates().GetLatestByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after upgrade")
	assert.Equal(t, rsu1.ClusterConfig.Name, rsu2.ClusterConfig.Name)

	// ensure license type still persisted and network filter still disabled after upgrade
	disabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(i.RuntimeID, upgradeClusterOperationID, &disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)

	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":15,
		"autoScalerMax":25,
		"maxSurge":13,
		"maxUnavailable":6
	}
}`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":14,
		"autoScalerMax":25,
		"maxSurge":13,
		"maxUnavailable":6
	}
}`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID = suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.AssertKymaResourceExists(upgradeOperationID)
	suite.AssertKymaLabelsExist(upgradeOperationID, map[string]string{
		"kyma-project.io/region":          "eastus",
		"kyma-project.io/platform-region": "cf-eu10",
	})
}

func TestClusterUpgradeUsesUpdatedAutoscalerParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// Create an SKR with a default autoscaler params
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com",
                        "sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    }
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "eastus"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// perform an update with custom autoscaler params
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":50,
		"autoScalerMax":80,
		"maxSurge":13,
		"maxUnavailable":9
	}
}`)
	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// when
	orchestrationResp := suite.CallAPI("POST", "upgrade/cluster",
		`{
				"strategy": {
				  "type": "parallel",
				  "schedule": "immediate",
				  "parallel": {
					"workers": 1
				  }
				},
				"dryRun": false,
				"targets": {
				  "include": [
					{
					  "subAccount": "sub-id"
					}
				  ]
				}
				}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)

	var upgradeKymaOperationID string
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Millisecond, 400*time.Millisecond, true, func(ctx context.Context) (bool, error) {
		var err error
		opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
		upgradeKymaOperationID, err = suite.DecodeLastUpgradeKymaOperationIDFromOrchestration(opResponse)
		return err == nil, nil
	})

	require.NoError(t, err)

	// then
	disabled := false
	suite.AssertShootUpgrade(upgradeKymaOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:   ptr.String("1.18"),
			MachineImage:        ptr.String("coreos"),
			MachineImageVersion: ptr.String("253"),

			MaxSurge:       ptr.Integer(13),
			MaxUnavailable: ptr.Integer(9),

			EnableKubernetesVersionAutoUpdate:   ptr.Bool(false),
			EnableMachineImageVersionAutoUpdate: ptr.Bool(false),

			OidcConfig:                    defaultOIDCConfig(),
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})

	suite.AssertKymaResourceExists(upgradeOperationID)
	suite.AssertKymaLabelsExist(upgradeOperationID, map[string]string{
		"kyma-project.io/region":          "eastus",
		"kyma-project.io/platform-region": "cf-eu10",
	})
}
