package configuration

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"

	"gopkg.in/yaml.v3"
)

type ProviderSpec struct {
	data dto
}

type regionDTO struct {
	DisplayName string   `yaml:"displayName"`
	Zones       []string `yaml:"zones"`
}

type providerDTO struct {
	Regions             map[string]regionDTO     `yaml:"regions"`
	MachineDisplayNames map[string]string        `yaml:"machines"`
	SupportingMachines  RegionsSupportingMachine `yaml:"regionsSupportingMachine,omitempty"`
	ZonesDiscovery      bool                     `yaml:"zonesDiscovery"`
	DualStack           bool                     `yaml:"dualStack,omitempty"`
}

type dto map[runtime.CloudProvider]providerDTO

func NewProviderSpecFromFile(filePath string) (*ProviderSpec, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Use the existing function to parse the specifications
	return NewProviderSpec(file)
}

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

func (p *ProviderSpec) AvailableZonesForAdditionalWorkers(machineType, region, providerType string) ([]string, error) {
	providerData := p.findProviderDTO(runtime.CloudProviderFromString(providerType))
	if providerData == nil {
		return []string{}, nil
	}

	if providerData.SupportingMachines == nil {
		return []string{}, nil
	}

	if !providerData.SupportingMachines.IsSupported(region, machineType) {
		return []string{}, nil
	}

	zones, err := providerData.SupportingMachines.AvailableZonesForAdditionalWorkers(machineType, region, providerType)
	if err != nil {
		return []string{}, fmt.Errorf("while getting available zones from regions supporting machine: %w", err)
	}

	return zones, nil
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

	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return nil
	}

	if regionData, ok := providerData.Regions[region]; ok {
		return &regionData
	}

	return nil
}

func (p *ProviderSpec) findProviderDTO(cp runtime.CloudProvider) *providerDTO {
	for name, provider := range p.data {
		// remove '-' to support "sap-converged-cloud" for CloudProvider SapConvergedCloud
		if strings.ToLower(strings.ReplaceAll(string(name), "-", "")) == strings.ToLower(string(cp)) {
			return &provider
		}
	}
	return nil
}

func (p *ProviderSpec) Validate(provider runtime.CloudProvider, region string) error {
	if dto := p.findRegion(provider, region); dto != nil {
		providerDTO := p.findProviderDTO(provider)
		if !providerDTO.ZonesDiscovery && (dto.Zones == nil || len(dto.Zones) == 0) {
			return fmt.Errorf("region %s for provider %s has no zones defined", region, provider)
		}
		if dto.DisplayName == "" {
			return fmt.Errorf("region %s for provider %s has no display name defined", region, provider)
		}
		return nil
	}
	return fmt.Errorf("region %s not found for provider %s", region, provider)
}

func (p *ProviderSpec) MachineDisplayNames(cp runtime.CloudProvider, machines []string) map[string]string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return nil
	}

	displayNames := map[string]string{}
	for _, machine := range machines {
		if displayName, ok := providerData.MachineDisplayNames[machine]; ok {
			displayNames[machine] = displayName
		} else {
			displayNames[machine] = machine // fallback to machine name if no display name is found
		}
	}
	return displayNames
}

func (p *ProviderSpec) RegionSupportingMachine(providerType string) (internal.RegionsSupporter, error) {
	providerData := p.findProviderDTO(runtime.CloudProviderFromString(providerType))
	if providerData == nil {
		return RegionsSupportingMachine{}, nil
	}

	if providerData.SupportingMachines == nil {
		return RegionsSupportingMachine{}, nil
	}
	return providerData.SupportingMachines, nil
}

func (p *ProviderSpec) ValidateZonesDiscovery() error {
	for provider, providerDTO := range p.data {
		if providerDTO.ZonesDiscovery {
			if provider != "aws" {
				return fmt.Errorf("zone discovery is not yet supported for the %s provider", provider)
			}

			for region, regionDTO := range providerDTO.Regions {
				if len(regionDTO.Zones) > 0 {
					slog.Warn(fmt.Sprintf("Provider %s has zones discovery enabled, but region %s is configured with %d static zone(s), which will be ignored.", provider, region, len(regionDTO.Zones)))
				}
			}

			for machineType, regionZones := range providerDTO.SupportingMachines {
				for region, zones := range regionZones {
					if len(zones) > 0 {
						slog.Warn(fmt.Sprintf("Provider %s has zones discovery enabled, but machine type %s in region %s is configured with %d static zone(s), which will be ignored.", provider, machineType, region, len(zones)))
					}
				}
			}
		}
	}

	return nil
}

func (p *ProviderSpec) ZonesDiscovery(cp runtime.CloudProvider) bool {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return false
	}
	return providerData.ZonesDiscovery
}

func (p *ProviderSpec) MachineTypes(cp runtime.CloudProvider) []string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return []string{}
	}

	machineTypes := make([]string, 0, len(providerData.MachineDisplayNames))
	for machineType := range providerData.MachineDisplayNames {
		machineTypes = append(machineTypes, machineType)
	}

	return machineTypes
}

func (p *ProviderSpec) Regions(cp runtime.CloudProvider) []string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return []string{}
	}

	regions := make([]string, 0, len(providerData.Regions))
	for region := range providerData.Regions {
		regions = append(regions, region)
	}

	sort.Strings(regions)
	return regions
}

func (p *ProviderSpec) IsDualStackSupported(cp runtime.CloudProvider) bool {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return false
	}
	return providerData.DualStack
}
