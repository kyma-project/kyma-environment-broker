package subscriptions

import (
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
)

type ParsedRule interface {
	Hyperscaler() string
	IsShared() bool
	IsEUAccess() bool
	Rule() string
}

// SecretBinding selector requirements
const (
	hyperscalerTypeReqFmt = gardener.HyperscalerTypeLabelKey + "=%s"
	tenantNameReqFmt      = gardener.TenantNameLabelKey + "=%s"

	dirtyReq    = gardener.DirtyLabelKey + "=true"
	internalReq = gardener.InternalLabelKey + "=true"
	sharedReq   = gardener.SharedLabelKey + "=true"
	euAccessReq = gardener.EUAccessLabelKey + "=true"

	notSharedReq = gardener.SharedLabelKey + `!=true`

	notDirtyReq       = `!` + gardener.DirtyLabelKey
	notInternalReq    = `!` + gardener.InternalLabelKey
	notEUAccessReq    = `!` + gardener.EUAccessLabelKey
	notTenantNamedReq = `!` + gardener.TenantNameLabelKey
)

type LabelSelectorBuilder struct {
	builder strings.Builder
	shared  bool
}

func NewLabelSelectorFromRuleset(rule ParsedRule) *LabelSelectorBuilder {
	selector := &LabelSelectorBuilder{builder: strings.Builder{}}
	selector.shared = rule.IsShared()
	selector.builder.WriteString(fmt.Sprintf(hyperscalerTypeReqFmt, rule.Hyperscaler()))
	if rule.IsEUAccess() {
		selector.with(euAccessReq)
	} else {
		selector.with(notEUAccessReq)
	}
	if rule.IsShared() {
		selector.with(sharedReq)
		return selector
	}
	selector.with(notSharedReq)
	selector.with(notDirtyReq)

	return selector
}

func (l *LabelSelectorBuilder) with(s string) {
	if l.builder.Len() == 0 {
		l.builder.WriteString(s)
		return
	}
	l.builder.WriteString("," + s)
}

func (l *LabelSelectorBuilder) BuildForTenantMatching(tenant string) string {
	if l.shared {
		return l.builder.String()
	}
	base := l.builder.String()
	tenantSelector := fmt.Sprintf(tenantNameReqFmt, tenant)
	return fmt.Sprintf("%s,%s", base, tenantSelector)
}

func (l *LabelSelectorBuilder) BuildAnySubscription() string {
	return l.builder.String()
}

func (l *LabelSelectorBuilder) BuildForSecretBindingClaim() string {
	base := l.builder.String()
	return fmt.Sprintf("%s,%s", base, notTenantNamedReq)
}
