package main

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
)

func TestClusterUpgrade_OneRuntimeHappyPath(t *testing.T) {
	// given
	suite := NewOrchestrationSuite(t, nil)
	defer suite.TearDown()
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationParams := fixOrchestrationParams(runtimeID)
	orchestrationID := suite.CreateUpgradeClusterOrchestration(orchestrationParams)

	suite.WaitForOrchestrationState(orchestrationID, orchestration.InProgress)

	// when
	suite.FinishUpgradeShootOperationByProvisioner(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, orchestration.Succeeded)

	suite.AssertShootUpgraded(runtimeID)
	suite.AssertShootNotUpgraded(otherRuntimeID)
}

func fixOrchestrationParams(runtimeID string) orchestration.Parameters {
	return orchestration.Parameters{
		Targets: orchestration.TargetSpec{
			Include: []orchestration.RuntimeTarget{
				{RuntimeID: runtimeID},
			},
		},
		Strategy: orchestration.StrategySpec{
			Type:     orchestration.ParallelStrategy,
			Schedule: time.Now().Format(time.RFC3339),
			Parallel: orchestration.ParallelStrategySpec{Workers: 1},
		},
		DryRun:     false,
		Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
	}
}
