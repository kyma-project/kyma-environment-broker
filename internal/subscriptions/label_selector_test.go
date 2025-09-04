package subscriptions

import (
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSelectNotShared(t *testing.T) {
	// given
	result := rules.Result{
		HyperscalerType: "aws",
		EUAccess:        false,
		Shared:          false,
		RawData: rules.RawData{
			Rule:   "",
			RuleNo: 0,
		},
	}

	selector := NewLabelSelectorFromRuleset(result)

	// when
	labels := selector.BuildForTenantMatching("tenant-a")
	labelsSBClaim := selector.BuildForSecretBindingClaim()
	labelsAnySubscription := selector.BuildAnySubscription()

	// then
	assert.Equal(t, "hyperscalerType=aws,!euAccess,!dirty,tenantName=tenant-a", labels)
	assert.Equal(t, "hyperscalerType=aws,!euAccess,!dirty,!tenantName,shared!=true", labelsSBClaim)
	assert.Equal(t, "hyperscalerType=aws,!euAccess,!dirty", labelsAnySubscription)
}

func TestSelectNotSharedEuAccess(t *testing.T) {
	// given
	result := rules.Result{
		HyperscalerType: "aws",
		EUAccess:        true,
		Shared:          false,
		RawData: rules.RawData{
			Rule:   "",
			RuleNo: 0,
		},
	}

	selector := NewLabelSelectorFromRuleset(result)

	// when
	labels := selector.BuildForTenantMatching("tenant-a")
	labelsSBClaim := selector.BuildForSecretBindingClaim()
	labelsAnySubscription := selector.BuildAnySubscription()

	// then
	assert.Equal(t, "hyperscalerType=aws,euAccess=true,!dirty,tenantName=tenant-a", labels)
	assert.Equal(t, "hyperscalerType=aws,euAccess=true,!dirty,!tenantName,shared!=true", labelsSBClaim)
	assert.Equal(t, "hyperscalerType=aws,euAccess=true,!dirty", labelsAnySubscription)
}

func TestSelectShared(t *testing.T) {
	// given
	result := rules.Result{
		HyperscalerType: "aws",
		EUAccess:        false,
		Shared:          true,
		RawData: rules.RawData{
			Rule:   "",
			RuleNo: 0,
		},
	}

	selector := NewLabelSelectorFromRuleset(result)

	// when
	labels := selector.BuildForTenantMatching("tenant-a")
	labelsAnySubscription := selector.BuildAnySubscription()

	// then
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared=true", labels)
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared=true", labelsAnySubscription)
}
