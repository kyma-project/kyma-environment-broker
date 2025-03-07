package rules

import (
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	content := `rule: 
  - aws                             
  - aws(PR=cf-eu11) -> EU           
  - azure                           
  - azure(PR=cf-ch20) -> EU         
  - gcp                            
  - gcp(PR=cf-sa30)                 
  - trial -> S                      `

	tmpfile, err := CreateTempFile(content)
	require.NoError(t, err)

	defer os.Remove(tmpfile)

	svc, err := NewRulesServiceFromFile(tmpfile, &broker.EnablePlans{"azure", "gcp", "trial", "aws"}, false, false, true)
	require.NoError(t, err)

	for _, result := range svc.Parsed.Results {
		require.False(t, result.HasErrors())
	}

	result := svc.Match(&ProvisioningAttributes{
		Plan:              "azure",
		PlatformRegion:    "cf-eu21",
		HyperscalerRegion: "eu-central-1",
	})

	require.NoError(t, err)

	found := false
	for _, matchingResult := range result {
		if matchingResult.FinalMatch {
			require.Equal(t, "azure", matchingResult.Rule.Plan)
			require.Equal(t, "", matchingResult.Rule.HyperscalerRegion)
			require.Equal(t, "", matchingResult.Rule.PlatformRegion)
			found = true
		}
	}

	require.True(t, found)
}
