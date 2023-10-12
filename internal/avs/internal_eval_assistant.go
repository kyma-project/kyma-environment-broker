package avs

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type InternalEvalAssistant struct {
	avsConfig   Config
	retryConfig *RetryConfig
}

func NewInternalEvalAssistant(avsConfig Config) *InternalEvalAssistant {
	return &InternalEvalAssistant{
		avsConfig:   avsConfig,
		retryConfig: &RetryConfig{maxTime: 10 * time.Minute, retryInterval: 30 * time.Second},
	}
}

func (iec *InternalEvalAssistant) CreateBasicEvaluationRequest(operations internal.Operation, url string) (*BasicEvaluationCreateRequest, error) {
	return newBasicEvaluationCreateRequest(operations, iec, url)
}

func (iec *InternalEvalAssistant) IsAlreadyCreated(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AvsEvaluationInternalId != 0
}

func (iec *InternalEvalAssistant) IsValid(lifecycleData internal.AvsLifecycleData) bool {
	return iec.IsAlreadyCreated(lifecycleData) && !iec.IsAlreadyDeletedOrEmpty(lifecycleData)
}

func (iec *InternalEvalAssistant) ProvideSuffix() string {
	return "int"
}

func (iec *InternalEvalAssistant) ProvideTesterAccessId(pp internal.ProvisioningParameters) int64 {
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialInternalTesterAccessId
	}
	return iec.avsConfig.InternalTesterAccessId
}

func (iec *InternalEvalAssistant) ProvideGroupId(pp internal.ProvisioningParameters) int64 {
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialGroupId
	}
	return iec.avsConfig.GroupId
}

func (iec *InternalEvalAssistant) ProvideParentId(pp internal.ProvisioningParameters) int64 {
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialParentId
	}
	return iec.avsConfig.ParentId
}

func (iec *InternalEvalAssistant) ProvideCheckType() string {
	return ""
}

func (iec *InternalEvalAssistant) ProvideTags(operation internal.Operation) []*Tag {

	tags := []*Tag{
		{
			Content:    operation.InstanceID,
			TagClassId: iec.avsConfig.InstanceIdTagClassId,
		},
		{
			Content:    operation.ProvisioningParameters.ErsContext.GlobalAccountID,
			TagClassId: iec.avsConfig.GlobalAccountIdTagClassId,
		},
		{
			Content:    operation.ProvisioningParameters.ErsContext.SubAccountID,
			TagClassId: iec.avsConfig.SubAccountIdTagClassId,
		},
		{
			Content:    operation.ProvisioningParameters.PlatformRegion,
			TagClassId: iec.avsConfig.LandscapeTagClassId,
		},
		{
			Content:    string(operation.ProvisioningParameters.PlatformProvider),
			TagClassId: iec.avsConfig.ProviderTagClassId,
		},
		{
			Content:    operation.ShootName,
			TagClassId: iec.avsConfig.ShootNameTagClassId,
		},
		{
			Content:    operation.ShootName,
			TagClassId: iec.avsConfig.GardenerShootNameTagClassId,
		},
	}

	region := ""
	if operation.ProvisioningParameters.Parameters.Region != nil {
		region = *operation.ProvisioningParameters.Parameters.Region
	} else if operation.LastRuntimeState.ClusterSetup != nil {
		region = operation.LastRuntimeState.ClusterSetup.Metadata.Region
	} else if operation.LastRuntimeState.ClusterConfig.Region != "" {
		region = operation.LastRuntimeState.ClusterConfig.Region
	} else if operation.Region != "" {
		region = operation.Region
	}

	tags = append(tags, &Tag{
		Content:    region,
		TagClassId: iec.avsConfig.RegionTagClassId,
	})

	return tags

}

func (iec *InternalEvalAssistant) ProvideNewOrDefaultServiceName(defaultServiceName string) string {
	if iec.avsConfig.InternalTesterService == "" {
		return defaultServiceName
	}
	return iec.avsConfig.InternalTesterService
}

func (iec *InternalEvalAssistant) SetEvalId(lifecycleData *internal.AvsLifecycleData, evalId int64) {
	lifecycleData.AvsEvaluationInternalId = evalId
}

func (iec *InternalEvalAssistant) SetEvalStatus(lifecycleData *internal.AvsLifecycleData, status string) {
	current := lifecycleData.AvsInternalEvaluationStatus.Current
	if ValidStatus(current) {
		lifecycleData.AvsInternalEvaluationStatus.Original = current
	}
	lifecycleData.AvsInternalEvaluationStatus.Current = status
}

func (iec *InternalEvalAssistant) GetEvalStatus(lifecycleData internal.AvsLifecycleData) string {
	return lifecycleData.AvsInternalEvaluationStatus.Current
}

func (iec *InternalEvalAssistant) GetOriginalEvalStatus(lifecycleData internal.AvsLifecycleData) string {
	return lifecycleData.AvsInternalEvaluationStatus.Original
}

func (iec *InternalEvalAssistant) IsInMaintenance(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AvsInternalEvaluationStatus.Current == StatusMaintenance
}

func (iec *InternalEvalAssistant) IsAlreadyDeletedOrEmpty(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSInternalEvaluationDeleted || lifecycleData.AvsEvaluationInternalId == 0
}

func (iec *InternalEvalAssistant) GetEvaluationId(lifecycleData internal.AvsLifecycleData) int64 {
	return lifecycleData.AvsEvaluationInternalId
}

func (iec *InternalEvalAssistant) SetDeleted(lifecycleData *internal.AvsLifecycleData, deleted bool) {
	lifecycleData.AVSInternalEvaluationDeleted = deleted
}

func (iec *InternalEvalAssistant) provideRetryConfig() *RetryConfig {
	return iec.retryConfig
}
