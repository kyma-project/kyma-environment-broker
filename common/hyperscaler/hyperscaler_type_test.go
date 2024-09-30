package hyperscaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHyperscalerTypeWithoutRegion(t *testing.T) {

	var testcases = []struct {
		testDescription string
		testType        Type
		expectedName    string
		expectedKey     string
		expectedRegion  string
	}{
		{"GCP Hyperscaler type without region",
			GCP(""), "gcp", "gcp", ""},
		{"AWS Hyperscaler type without region",
			AWS(), "aws", "aws", ""},
		{"Azure Hyperscaler type without region",
			Azure(), "azure", "azure", ""},
	}
	for _, testcase := range testcases {

		t.Run(testcase.testDescription, func(t *testing.T) {
			assert.Equal(t, testcase.expectedName, testcase.testType.GetName())
			assert.Equal(t, testcase.expectedKey, testcase.testType.GetKey())
			assert.Equal(t, testcase.expectedRegion, testcase.testType.GetRegion())
		})
	}
}

func TestSapConvergedCloudHyperscalerTypeWithRegion(t *testing.T) {
	testHypType := SapConvergedCloud("eu-de-test")
	assert.Equal(t, "openstack", testHypType.GetName())
	assert.Equal(t, "openstack_eu-de-test", testHypType.GetKey())
	assert.Equal(t, "eu-de-test", testHypType.GetRegion())
}

func TestGCPHyperscalerTypeWithPlatformRegion(t *testing.T) {
	t.Run("KSA platform region", func(t *testing.T) {
		testHypType := GCP("cf-sa30")
		assert.Equal(t, "gcp", testHypType.GetName())
		assert.Equal(t, "gcp_cf-sa30", testHypType.GetKey())
		assert.Equal(t, "", testHypType.GetRegion())
	})

	t.Run("platform region other than KSA", func(t *testing.T) {
		testHypType := GCP("cf-jp30")
		assert.Equal(t, "gcp", testHypType.GetName())
		assert.Equal(t, "gcp", testHypType.GetKey())
		assert.Equal(t, "", testHypType.GetRegion())
	})
}
