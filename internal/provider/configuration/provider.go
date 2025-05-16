package configuration

import (
	"fmt"
	"io"
	"math/rand"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"gopkg.in/yaml.v2"
)

type ProviderSpec struct {
	data dto
}

type regionDTO struct {
	DisplayName string   `yaml:"displayName"`
	Zones       []string `yaml:"zones"`
}

type providerDTO struct {
	Regions map[string]regionDTO `yaml:"regions"`
}

type dto map[runtime.CloudProvider]providerDTO

func NewProviderSpec(r io.Reader) (*ProviderSpec, error) {
	data := &dto{}
	d := yaml.NewDecoder(r)
	err := d.Decode(data)
	return &ProviderSpec{
		data: *data,
	}, err
}

func (p *ProviderSpec) RegionDisplayName(cp runtime.CloudProvider, region string) string {
	dto := p.findRegion(cp, region)
	if dto == nil {
		return region
	}
	return dto.DisplayName
}

func (p *ProviderSpec) RegionDisplayNames(cp runtime.CloudProvider, regions []string) map[string]string {
	displayNames := map[string]string{}
	for _, region := range regions {
		r := p.findRegion(cp, region)
		if r == nil {
			displayNames[region] = region
			continue
		}
		displayNames[region] = r.DisplayName
	}
	return displayNames
}

func (p *ProviderSpec) Zones(cp runtime.CloudProvider, region string) []string {
	dto := p.findRegion(cp, region)
	if dto == nil {
		return []string{}
	}
	return dto.Zones
}

func (p *ProviderSpec) RandomZones(cp runtime.CloudProvider, region string, zonesCount int) []string {
	availableZones := p.Zones(cp, region)
	rand.Shuffle(len(availableZones), func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })
	if zonesCount > len(availableZones) {
		// get maximum number of zones for region
		zonesCount = len(availableZones)
	}

	return availableZones[:zonesCount]
}

func (p *ProviderSpec) findRegion(cp runtime.CloudProvider, region string) *regionDTO {
	for name, provider := range p.data {
		// remove '-' to support "sap-converged-cloud" for CloudProvider SapConvergedCloud
		if strings.ToLower(strings.ReplaceAll(string(name), "-", "")) != strings.ToLower(string(cp)) {
			continue
		}
		if regionData, ok := provider.Regions[region]; ok {
			return &regionData
		}
	}
	return nil
}

func (p *ProviderSpec) Validate(provider runtime.CloudProvider, region string) error {
	if dto := p.findRegion(provider, region); dto != nil {
		if dto.Zones == nil || len(dto.Zones) == 0 {
			return fmt.Errorf("region %s for provider %s has no zones defined", region, provider)
		}
		if dto.DisplayName == "" {
			return fmt.Errorf("region %s for provider %s has no display name defined", region, provider)
		}
		return nil
	}
	return fmt.Errorf("region %s not found for provider %s", region, provider)
}
