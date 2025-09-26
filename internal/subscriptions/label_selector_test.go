package subscriptions_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"
	"github.com/stretchr/testify/assert"
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

	selector := subscriptions.NewLabelSelectorFromRuleset(result)

	// when
	labels := selector.BuildForTenantMatching("tenant-a")
	labelsSBClaim := selector.BuildForSecretBindingClaim()
	labelsAnySubscription := selector.BuildAnySubscription()

	// then
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared!=true,!dirty,tenantName=tenant-a", labels)
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared!=true,!dirty,!tenantName", labelsSBClaim)
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared!=true,!dirty", labelsAnySubscription)
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

	selector := subscriptions.NewLabelSelectorFromRuleset(result)

	// when
	labels := selector.BuildForTenantMatching("tenant-a")
	labelsSBClaim := selector.BuildForSecretBindingClaim()
	labelsAnySubscription := selector.BuildAnySubscription()

	// then
	assert.Equal(t, "hyperscalerType=aws,euAccess=true,shared!=true,!dirty,tenantName=tenant-a", labels)
	assert.Equal(t, "hyperscalerType=aws,euAccess=true,shared!=true,!dirty,!tenantName", labelsSBClaim)
	assert.Equal(t, "hyperscalerType=aws,euAccess=true,shared!=true,!dirty", labelsAnySubscription)
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

	selector := subscriptions.NewLabelSelectorFromRuleset(result)

	// when
	labels := selector.BuildForTenantMatching("tenant-a")
	labelsAnySubscription := selector.BuildAnySubscription()

	// then
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared=true", labels)
	assert.Equal(t, "hyperscalerType=aws,!euAccess,shared=true", labelsAnySubscription)
}
