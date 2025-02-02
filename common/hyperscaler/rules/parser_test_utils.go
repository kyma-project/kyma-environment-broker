package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ParserHappyPathTest(t *testing.T, parser Parser) {

    t.Run("with plan", func(t *testing.T) {
        rule, err := parser.Parse("azure")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.PlatformRegion)
        require.Empty(t, rule.HyperscalerRegion)

        require.Equal(t, false, rule.EuAccess)
        require.Equal(t, false, rule.Shared)
    })

    t.Run("with plan and single input attribute", func(t *testing.T) {
        rule, err := parser.Parse("azure(PR=westeurope)")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.PlatformRegion)
        require.Empty(t, rule.HyperscalerRegion)

        require.Equal(t, false, rule.EuAccess)
        require.Equal(t, false, rule.Shared)

        rule, err = parser.Parse("azure(HR=westeurope)")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.Equal(t, false, rule.EuAccess)
        require.Equal(t, false, rule.Shared)
    })

    t.Run("with plan all output attributes - different positions", func(t *testing.T) {
        rule, err := parser.Parse("azure(PR=easteurope,HR=westeurope)")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.HyperscalerRegion)
        require.Equal(t, "easteurope", rule.PlatformRegion)

        require.False(t, rule.EuAccess)
        require.False(t, rule.Shared)

        rule, err = parser.Parse("azure(HR=westeurope,PR=easteurope)")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "westeurope", rule.HyperscalerRegion)
        require.Equal(t, "easteurope", rule.PlatformRegion)

        require.False(t, rule.EuAccess)
        require.False(t, rule.Shared)
    })

    t.Run("with plan and single output attribute", func(t *testing.T) {
        rule, err := parser.Parse("azure->S")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.False(t, rule.EuAccess)
        require.True(t, rule.Shared)

        rule, err = parser.Parse("azure->EU")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.True(t, rule.EuAccess)
        require.False(t, rule.Shared)
    })


    t.Run("with plan and all output attributes - different positions", func(t *testing.T) {
        rule, err := parser.Parse("azure->S,EU")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.True(t, rule.EuAccess)
        require.True(t, rule.Shared)

        rule, err = parser.Parse("azure->EU,S")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Empty(t, rule.PlatformRegion)

        require.True(t, rule.EuAccess)
        require.True(t, rule.Shared)
    })

    t.Run("with plan and single output/input attributes", func(t *testing.T) {
        rule, err := parser.Parse("azure(PR=westeurope)->EU")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Empty(t, rule.HyperscalerRegion)
        require.Equal(t, "westeurope", rule.PlatformRegion)

        require.True(t, rule.EuAccess)
        require.False(t, rule.Shared)
    }) 

    t.Run("with plan and all input/output attributes", func(t *testing.T) {
        rule, err := parser.Parse("azure(PR=westeurope, HR=easteurope)->EU,S")
        require.NoError(t, err)

        require.NotNil(t, rule)
        require.Equal(t, "azure", rule.Plan)
        require.Equal(t, "easteurope", rule.HyperscalerRegion)
        require.Equal(t, "westeurope", rule.PlatformRegion)

        require.True(t, rule.EuAccess)
        require.True(t, rule.Shared)
    }) 
}


func ParserValidationTest(t *testing.T, parser Parser) {

    t.Run("with paranthesis only", func(t *testing.T) {
        rule, err := parser.Parse("()")
        require.Nil(t, rule)
        require.Error(t, err)
    })

    t.Run("with arrow only", func(t *testing.T) {
        rule, err := parser.Parse("->")
        require.Nil(t, rule)
        require.Error(t, err)
    })

    t.Run("with incorrect attributes list", func(t *testing.T) {
        rule, err := parser.Parse("test(,)->,")
        require.Nil(t, rule)
        require.Error(t, err)
    
        rule, err = parser.Parse("test(PR=west,HR=east)->,")
        require.Nil(t, rule)
        require.Error(t, err)
    })

//     t.Run("with duplicated input attribute", func(t *testing.T) {
//         rule, err := parser.Parse("azure(PR=test,PR=test2)")
//         require.Nil(t, rule)
//         require.Error(t, err)
    
//         rule, err = parser.Parse("test(PR=west,HR=east)->EU,EU")
//         require.Nil(t, rule)
//         require.Error(t, err)
//     })
}