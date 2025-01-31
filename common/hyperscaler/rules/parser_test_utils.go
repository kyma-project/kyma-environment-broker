package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ParserTest(t *testing.T, parser Parser) {

    t.Run("with plan", func(t *testing.T) {
        rule := parser.Parse("azure")

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.PlatformRegion)
        require.Empty(t, rule.HyperscalerRegion)

        require.Equal(t, false, rule.EuAccess)
        require.Equal(t, false, rule.Shared)
    })

    t.Run("with plan and platform region", func(t *testing.T) {
        rule := parser.Parse("azure(PR=westeurope)")

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.PlatformRegion)
        require.Empty(t, rule.HyperscalerRegion)

        require.Equal(t, false, rule.EuAccess)
        require.Equal(t, false, rule.Shared)
    })

    t.Run("with plan and hyperscaler region", func(t *testing.T) {
        rule := parser.Parse("azure(HR=westeurope)")

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.Equal(t, false, rule.EuAccess)
        require.Equal(t, false, rule.Shared)
    })

    t.Run("with plan, platform and hyperscaler region", func(t *testing.T) {
        rule := parser.Parse("azure(PR=easteurope, HR=westeurope)")

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.HyperscalerRegion)
        require.Equal(t, "easteurope", rule.PlatformRegion)

        require.False(t, rule.EuAccess)
        require.False(t, rule.Shared)
    })

    t.Run("with plan and shared", func(t *testing.T) {
        rule := parser.Parse("azure->S")

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.False(t, rule.EuAccess)
        require.True(t, rule.Shared)
    })


    t.Run("with plan, shared and euAccess", func(t *testing.T) {
        rule := parser.Parse("azure->S,EU")

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.True(t, rule.EuAccess)
        require.True(t, rule.Shared)
    })

}