package configuration

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"regexp"
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
	MachinesVersions    map[string]string        `yaml:"machinesVersions,omitempty"`
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
		if strings.EqualFold(strings.ReplaceAll(string(name), "-", ""), string(cp)) {
			return &provider
		}
	}
	return nil
}

func (p *ProviderSpec) Validate(provider runtime.CloudProvider, region string) error {
	if dto := p.findRegion(provider, region); dto != nil {
		providerDTO := p.findProviderDTO(provider)
		if !providerDTO.ZonesDiscovery && len(dto.Zones) == 0 {
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

// ResolveMachineType resolves a given machine type to its versioned equivalent
// using provider-specific template mappings. If no templates match, or if no
// provider data is available, the original machineType is returned unchanged.
func (p *ProviderSpec) ResolveMachineType(cp runtime.CloudProvider, machineType string) string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil || len(providerData.MachinesVersions) == 0 {
		return machineType
	}

	// Sort templates from the most specific to the least specific, so we prefer
	// the most constrained match when multiple templates could match the input.
	templates := make([]string, 0, len(providerData.MachinesVersions))
	for inputTemplate := range providerData.MachinesVersions {
		templates = append(templates, inputTemplate)
	}

	sort.SliceStable(templates, func(i, j int) bool {
		leftPlaceholderCount := templatePlaceholderCount(templates[i])
		rightPlaceholderCount := templatePlaceholderCount(templates[j])

		if leftPlaceholderCount != rightPlaceholderCount {
			return leftPlaceholderCount < rightPlaceholderCount
		}

		leftLiteralLength := templateLiteralLength(templates[i])
		rightLiteralLength := templateLiteralLength(templates[j])

		if leftLiteralLength != rightLiteralLength {
			return leftLiteralLength > rightLiteralLength
		}

		return templates[i] < templates[j]
	})

	// Resolve the first matching template.
	for _, inputTemplate := range templates {
		outputTemplate := providerData.MachinesVersions[inputTemplate]

		regex, placeholderNames := templateToRegex(inputTemplate)

		// regex.FindStringSubmatch returns either nil (no match) or a slice where:
		// matchedValues[0] is the full match and matchedValues[1:] are the capture groups.
		// Since templateToRegex creates exactly one capture group per placeholder,
		// we expect len(matchedValues) == len(placeholderNames) + 1 here.
		matchedValues := regex.FindStringSubmatch(machineType)
		if matchedValues == nil {
			continue
		}

		// Build placeholder -> captured value map.
		values := make(map[string]string, len(placeholderNames))
		for i, name := range placeholderNames {
			values[name] = matchedValues[i+1]
		}

		// Replace placeholders in the output template with captured values.
		resolved := replaceTemplatePlaceholders(outputTemplate, values)
		if resolved != "" {
			return resolved
		}
	}

	// No template matched, so return the original value unchanged.
	return machineType
}

var placeholderRegex = regexp.MustCompile(`{(\w+)}`)

// templatePlaceholderCount returns the number of placeholders in the template.
// Fewer placeholders means a more specific template.
func templatePlaceholderCount(template string) int {
	return len(placeholderRegex.FindAllString(template, -1))
}

// templateLiteralLength returns the number of literal characters in the template.
// More literal characters means a more specific template.
func templateLiteralLength(template string) int {
	return len(placeholderRegex.ReplaceAllString(template, ""))
}

// templateToRegex converts a template such as "m.{size}" into a regular expression
// like "^m\\.([^.]+)$" and returns the placeholder names in capture-group order.
func templateToRegex(template string) (*regexp.Regexp, []string) {
	var pattern strings.Builder
	pattern.WriteString("^")

	placeholderNames := make([]string, 0)
	lastIdx := 0

	for _, match := range placeholderRegex.FindAllStringSubmatchIndex(template, -1) {
		fullStart, fullEnd := match[0], match[1]
		nameStart, nameEnd := match[2], match[3]

		// Escape literal text between placeholders.
		pattern.WriteString(regexp.QuoteMeta(template[lastIdx:fullStart]))

		placeholderName := template[nameStart:nameEnd]
		placeholderNames = append(placeholderNames, placeholderName)

		// Placeholder values in current machine-type templates are alphanumeric.
		// This prevents matching already-versioned values such as:
		// Standard_D48s_v5 against Standard_D{size}
		pattern.WriteString(`([a-zA-Z0-9]+)`)

		lastIdx = fullEnd
	}

	// Escape trailing literal text.
	pattern.WriteString(regexp.QuoteMeta(template[lastIdx:]))
	pattern.WriteString("$")

	return regexp.MustCompile(pattern.String()), placeholderNames
}

// replaceTemplatePlaceholders replaces placeholders in the template with resolved values.
// Unknown placeholders are left unchanged.
func replaceTemplatePlaceholders(template string, values map[string]string) string {
	return placeholderRegex.ReplaceAllStringFunc(template, func(token string) string {
		name := token[1 : len(token)-1]
		if value, ok := values[name]; ok {
			return value
		}
		return token
	})
}
