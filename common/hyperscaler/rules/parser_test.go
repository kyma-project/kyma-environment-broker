package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParserHappyPath(t *testing.T) {

	t.Run("with plan", func(t *testing.T) {
		parser := &SimpleParser{}

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
		parser := &SimpleParser{}
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
		parser := &SimpleParser{}
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
		parser := &SimpleParser{}
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
		parser := &SimpleParser{}
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
		parser := &SimpleParser{}
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
		parser := &SimpleParser{}
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

func TestParserValidation(t *testing.T) {

	parser := &SimpleParser{}

	t.Run("with parenthesis only, no attributes", func(t *testing.T) {
		rule, err := parser.Parse("()")
		require.Nil(t, rule)
		require.Error(t, err)
	})

	t.Run("with duplicated arrow", func(t *testing.T) {
		rule, err := parser.Parse("->->")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("plan(PR=test, HR=test2) ->-> S, EU")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("plan(PR=test, HR=test2)-> S ->  EU")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("plan->(PR=test, HR=test2)-> S, EU")
		require.Nil(t, rule)
		require.Error(t, err)

	})

	t.Run("with duplicated or unclosed parantheses", func(t *testing.T) {
		rule, err := parser.Parse("(())")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("(")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("((")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse(")")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("))")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("())")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws(())")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws(")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws((")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws))")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws())")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("aws(PR=westeu, HR=easteu")
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

	t.Run("with duplicated input, output attribute - PR", func(t *testing.T) {
		rule, err := parser.Parse("azure(PR=test,PR=test2)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(PR=test,PR)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(HR=test,HR=test2)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(HR=test,HR)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west,HR=east)->EU,EU")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west,HR=east)->S,S")
		require.Nil(t, rule)
		require.Error(t, err)

	})

	t.Run("with whitespace which should be ignored", func(t *testing.T) {
		rule, err := parser.Parse("azure        (       PR              =     TEST, HR   =            TEST2                   ) ->          EU              ,             S")
		require.NoError(t, err)

		require.NotNil(t, rule)
		require.Equal(t, "azure", rule.Plan)
		require.Equal(t, "TEST", rule.PlatformRegion)
		require.Equal(t, "TEST2", rule.HyperscalerRegion)

		require.True(t, rule.EuAccess)
		require.True(t, rule.Shared)
	})

	t.Run("with invalid comma", func(t *testing.T) {
		rule, err := parser.Parse("azure(PR=test,,PR=test2)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(,,)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west,HR=east)->,,")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west,HR=east)->,S")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west,,)->S,EU")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west,HR=east)->S,,")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("test(PR=west.HR=east)->S")
		require.Nil(t, rule)
		require.Error(t, err)

	})

	t.Run("with unsupported attributes", func(t *testing.T) {
		rule, err := parser.Parse("azure(DR=invalid-key)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(PR=valid-key)->S,AA")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(PR=valid-key)->AA")
		require.Nil(t, rule)
		require.Error(t, err)
	})

	t.Run("without input attribute value", func(t *testing.T) {
		rule, err := parser.Parse("azure(PR)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(PR=)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(PR, HR)")
		require.Nil(t, rule)
		require.Error(t, err)

		rule, err = parser.Parse("azure(HR)")
		require.Nil(t, rule)
		require.Error(t, err)
	})
}
