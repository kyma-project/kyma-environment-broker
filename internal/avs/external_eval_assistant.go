package avs

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

const externalEvalCheckType = "HTTPSGET"

type ExternalEvalAssistant struct {
	avsConfig   Config
	retryConfig *RetryConfig
}

func NewExternalEvalAssistant(avsConfig Config) *ExternalEvalAssistant {
	return &ExternalEvalAssistant{
		avsConfig:   avsConfig,
		retryConfig: &RetryConfig{maxTime: 20 * time.Minute, retryInterval: 30 * time.Second},
	}
}

func (eea *ExternalEvalAssistant) CreateBasicEvaluationRequest(operations internal.Operation, url string) (*BasicEvaluationCreateRequest, error) {
	return newBasicEvaluationCreateRequest(operations, eea, url)
}

func (eea *ExternalEvalAssistant) IsAlreadyCreated(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSEvaluationExternalId != 0
}

func (eea *ExternalEvalAssistant) IsValid(lifecycleData internal.AvsLifecycleData) bool {
	return eea.IsAlreadyCreated(lifecycleData) && !eea.IsAlreadyDeletedOrEmpty(lifecycleData)
}

func (eea *ExternalEvalAssistant) ProvideSuffix() string {
	return "ext"
}

func (eea *ExternalEvalAssistant) ProvideTesterAccessId(_ internal.ProvisioningParameters) int64 {
	return eea.avsConfig.ExternalTesterAccessId
}

func (eea *ExternalEvalAssistant) ProvideGroupId(_ internal.ProvisioningParameters) int64 {
	return eea.avsConfig.GroupId
}

func (eea *ExternalEvalAssistant) ProvideParentId(_ internal.ProvisioningParameters) int64 {
	return eea.avsConfig.ParentId
}

// func (iec *InternalEvalAssistant) ProvideTags(pp internal.ProvisioningParameters, rt runtime.RuntimeDTO) []*Tag {
func (eea *ExternalEvalAssistant) ProvideTags(pp internal.ProvisioningParameters) []*Tag {
	var Tags []*Tag
	Tags = append(Tags, &Tag{
		Content:    "rt.InstanceID",
		TagClassId: eea.avsConfig.InstanceIdTagClassId,
	})

	Tags = append(Tags, &Tag{
		Content:    pp.ErsContext.GlobalAccountID,
		TagClassId: eea.avsConfig.GlobalAccountIdTagClassId,
	})

	Tags = append(Tags, &Tag{
		Content:    pp.ErsContext.SubAccountID,
		TagClassId: eea.avsConfig.SubAccountIdTagClassId,
	})

	Tags = append(Tags, &Tag{
		Content:    "rt.SubAccountRegion",
		TagClassId: eea.avsConfig.LandscapeTagClassId,
	})

	Tags = append(Tags, &Tag{
		// this returns with panic: panic: runtime error: invalid memory address or nil pointer dereference [recovered]
		// am I referencing it wrong?
		Content:    "string(*pp.Parameters.Provider)",
		TagClassId: eea.avsConfig.ProviderTagClassId,
	})

	Tags = append(Tags, &Tag{
		Content:    pp.PlatformRegion,
		TagClassId: eea.avsConfig.RegionTagClassId,
	})

	Tags = append(Tags, &Tag{
		Content:    pp.Parameters.ShootName,
		TagClassId: eea.avsConfig.ShootNameTagClassId,
	})

	return Tags
}

func (eea *ExternalEvalAssistant) ProvideNewOrDefaultServiceName(defaultServiceName string) string {
	if eea.avsConfig.ExternalTesterService == "" {
		return defaultServiceName
	}
	return eea.avsConfig.ExternalTesterService
}

func (eea *ExternalEvalAssistant) SetEvalId(lifecycleData *internal.AvsLifecycleData, evalId int64) {
	lifecycleData.AVSEvaluationExternalId = evalId
}

func (eea *ExternalEvalAssistant) SetEvalStatus(lifecycleData *internal.AvsLifecycleData, status string) {
	current := lifecycleData.AvsExternalEvaluationStatus.Current
	if ValidStatus(current) {
		lifecycleData.AvsExternalEvaluationStatus.Original = current
	}
	lifecycleData.AvsExternalEvaluationStatus.Current = status
}

func (eea *ExternalEvalAssistant) GetEvalStatus(lifecycleData internal.AvsLifecycleData) string {
	return lifecycleData.AvsExternalEvaluationStatus.Current
}

func (eea *ExternalEvalAssistant) GetOriginalEvalStatus(lifecycleData internal.AvsLifecycleData) string {
	return lifecycleData.AvsExternalEvaluationStatus.Original
}

func (eea *ExternalEvalAssistant) IsInMaintenance(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AvsExternalEvaluationStatus.Current == StatusMaintenance
}

func (eea *ExternalEvalAssistant) ProvideCheckType() string {
	return externalEvalCheckType
}

func (eea *ExternalEvalAssistant) IsAlreadyDeletedOrEmpty(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSExternalEvaluationDeleted || lifecycleData.AVSEvaluationExternalId == 0
}

func (eea *ExternalEvalAssistant) GetEvaluationId(lifecycleData internal.AvsLifecycleData) int64 {
	return lifecycleData.AVSEvaluationExternalId
}

func (eea *ExternalEvalAssistant) SetDeleted(lifecycleData *internal.AvsLifecycleData, deleted bool) {
	lifecycleData.AVSExternalEvaluationDeleted = deleted
}

func (eea *ExternalEvalAssistant) provideRetryConfig() *RetryConfig {
	return eea.retryConfig
}
