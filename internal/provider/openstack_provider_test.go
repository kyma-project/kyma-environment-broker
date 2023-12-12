package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
)

func TestZonesForOpenStackZones(t *testing.T) {
	regions := broker.SapConvergedCloudRegions()
	for _, region := range regions {
		_, exists := openstackZones[region]
		assert.True(t, exists)
	}
	_, exists := openstackZones[DefaultSapConvergedCloudRegion]
	assert.True(t, exists)
}
