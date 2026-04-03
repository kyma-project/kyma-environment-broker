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
	"unicode"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"

	"gopkg.in/yaml.v3"
)

type ProviderSpec struct {
	data dto
}

type dto map[runtime.CloudProvider]providerDTO

type providerDTO struct {
	Regions                  map[string]regionDTO           `yaml:"regions"`
	MachineDisplayNames      map[string]string              `yaml:"machines"`
	RegionsSupportingMachine map[string]map[string][]string `yaml:"regionsSupportingMachine,omitempty"`
	ZonesDiscovery           bool                           `yaml:"zonesDiscovery"`
	DualStack                bool                           `yaml:"dualStack,omitempty"`
	MachinesVersions         map[string]string              `yaml:"machinesVersions,omitempty"`
}

type regionDTO struct {
	DisplayName string   `yaml:"displayName"`
	Zones       []string `yaml:"zones"`
}

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

			for machineType, regionZones := range providerDTO.RegionsSupportingMachine {
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

func (p *ProviderSpec) IsRegionSupported(cp runtime.CloudProvider, region, machineType string) bool {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return true
	}

	machineType = p.ResolveMachineType(cp, machineType)

	for machineFamily, regions := range providerData.RegionsSupportingMachine {
		// Keep in mind that machineType should match at most one machineFamily
		if strings.HasPrefix(machineType, machineFamily) {
			if _, exists := regions[region]; exists {
				return true
			}
			return false
		}
	}

	return true
}

func (p *ProviderSpec) SupportedRegions(cp runtime.CloudProvider, machineType string) []string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return []string{}
	}

	machineType = p.ResolveMachineType(cp, machineType)

	for machineFamily, regionsMap := range providerData.RegionsSupportingMachine {
		// Keep in mind that machineType should match at most one machineFamily
		if strings.HasPrefix(machineType, machineFamily) {
			regions := make([]string, 0, len(regionsMap))
			for region := range regionsMap {
				regions = append(regions, region)
			}
			sort.Strings(regions)
			return regions
		}
	}

	return []string{}
}

func (p *ProviderSpec) AvailableZones(cp runtime.CloudProvider, machineType, region string) []string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil {
		return []string{}
	}

	machineType = p.ResolveMachineType(cp, machineType)

	for machineFamily, regionsMap := range providerData.RegionsSupportingMachine {
		// Keep in mind that machineType should match at most one machineFamily
		if strings.HasPrefix(machineType, machineFamily) {
			zones := regionsMap[region]
			if len(zones) == 0 {
				return []string{}
			}
			rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
			return zones
		}
	}

	return []string{}
}

func (p *ProviderSpec) ValidateMachinesVersions() error {
	var errs []error

	for provider, providerDTO := range p.data {
		if len(providerDTO.MachinesVersions) == 0 {
			continue
		}

		for inputTemplate, outputTemplate := range providerDTO.MachinesVersions {
			errs = append(errs, validateMachinesVersionMapping(provider, inputTemplate, outputTemplate)...)
		}
	}

	if len(errs) > 0 {
		var errMses []string
		for _, err := range errs {
			errMses = append(errMses, err.Error())
		}

		return fmt.Errorf("Failed to validate machines versions: %s", strings.Join(errMses, "; "))
	}

	return nil
}

func validateMachinesVersionMapping(providerName runtime.CloudProvider, inputTemplate, outputTemplate string) []error {
	var errs []error

	if strings.TrimSpace(inputTemplate) == "" {
		errs = append(errs, fmt.Errorf("provider %q: machinesVersions contains an empty input template", providerName))
		return errs
	}
	if strings.TrimSpace(outputTemplate) == "" {
		errs = append(errs, fmt.Errorf("provider %q: machinesVersions[%q] has an empty output template", providerName, inputTemplate))
		return errs
	}

	inputPlaceholders, inputErrs := parseAndValidateTemplatePlaceholders(inputTemplate, true)
	for _, err := range inputErrs {
		errs = append(errs, fmt.Errorf("provider %q: invalid input template %q: %w", providerName, inputTemplate, err))
	}

	outputPlaceholders, outputErrs := parseAndValidateTemplatePlaceholders(outputTemplate, false)
	for _, err := range outputErrs {
		errs = append(errs, fmt.Errorf("provider %q: invalid output template %q for input %q: %w", providerName, outputTemplate, inputTemplate, err))
	}

	if len(inputErrs) > 0 || len(outputErrs) > 0 {
		return errs
	}

	inputSet := make(map[string]struct{}, len(inputPlaceholders))
	for _, name := range inputPlaceholders {
		inputSet[name] = struct{}{}
	}

	for _, name := range outputPlaceholders {
		if _, ok := inputSet[name]; !ok {
			errs = append(errs, fmt.Errorf(
				"provider %q: invalid mapping %q -> %q: output placeholder %q is not defined in input template",
				providerName, inputTemplate, outputTemplate, "{"+name+"}",
			))
		}
	}

	return errs
}

func parseAndValidateTemplatePlaceholders(template string, rejectAdjacent bool) ([]string, []error) {
	var errs []error
	var names []string

	seen := make(map[string]struct{})
	var prevWasPlaceholder bool

	for i := 0; i < len(template); {
		switch template[i] {
		case '{':
			end := strings.IndexByte(template[i+1:], '}')
			if end == -1 {
				errs = append(errs, fmt.Errorf("unclosed placeholder starting at position %d", i))
				return names, errs
			}

			end += i + 1
			name := template[i+1 : end]

			if name == "" {
				errs = append(errs, fmt.Errorf("empty placeholder at position %d", i))
			} else if !isValidPlaceholderName(name) {
				errs = append(errs, fmt.Errorf("invalid placeholder name %q at position %d", name, i))
			} else {
				if _, exists := seen[name]; exists {
					errs = append(errs, fmt.Errorf("duplicate placeholder %q", "{"+name+"}"))
				}
				seen[name] = struct{}{}
				names = append(names, name)
			}

			if rejectAdjacent && prevWasPlaceholder {
				errs = append(errs, fmt.Errorf("adjacent placeholders are not allowed"))
			}

			prevWasPlaceholder = true
			i = end + 1

		case '}':
			errs = append(errs, fmt.Errorf("unmatched closing brace at position %d", i))
			i++

		default:
			prevWasPlaceholder = false
			i++
		}
	}

	return names, errs
}

func isValidPlaceholderName(name string) bool {
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ResolveMachineType resolves a given machine type to its versioned equivalent
// using provider-specific template mappings.
//
// Example:
//
//	input:  "Standard_D4"
//	template: "Standard_D{size}" -> "Standard_D{size}_v3"
//	output: "Standard_D4_v3"
//
// If no templates match, or if no provider data is available,
// the original machineType is returned unchanged.
func (p *ProviderSpec) ResolveMachineType(cp runtime.CloudProvider, machineType string) string {
	providerData := p.findProviderDTO(cp)
	if providerData == nil || len(providerData.MachinesVersions) == 0 {
		return machineType
	}

	// Collect all input templates (keys of the mapping)
	templates := make([]string, 0, len(providerData.MachinesVersions))
	for inputTemplate := range providerData.MachinesVersions {
		templates = append(templates, inputTemplate)
	}

	// Sort templates by "specificity" so that the best match is tried first.
	// Rules:
	//   1. Fewer placeholders → more specific
	//   2. More literal characters → more specific
	//   3. Lexicographic order (stable fallback)
	sort.SliceStable(templates, func(i, j int) bool {
		leftPlaceholderCount := templatePlaceholderCount(templates[i])
		rightPlaceholderCount := templatePlaceholderCount(templates[j])

		// Prefer templates with fewer placeholders
		if leftPlaceholderCount != rightPlaceholderCount {
			return leftPlaceholderCount < rightPlaceholderCount
		}

		leftLiteralLength := templateLiteralLength(templates[i])
		rightLiteralLength := templateLiteralLength(templates[j])

		// Prefer templates with more fixed (non-placeholder) characters
		if leftLiteralLength != rightLiteralLength {
			return leftLiteralLength > rightLiteralLength
		}

		// Final deterministic ordering
		return templates[i] < templates[j]
	})

	// Try to resolve using the first matching (most specific) template
	for _, inputTemplate := range templates {
		outputTemplate := providerData.MachinesVersions[inputTemplate]

		// Convert template into regex + ordered placeholder names
		regex, placeholderNames := templateToRegex(inputTemplate)

		// regex.FindStringSubmatch returns either nil (no match) or a slice where:
		// matchedValues[0] is the full match and matchedValues[1:] are the capture groups.
		// Since templateToRegex creates exactly one capture group per placeholder,
		// we expect len(matchedValues) == len(placeholderNames) + 1 here.
		matchedValues := regex.FindStringSubmatch(machineType)
		if matchedValues == nil {
			continue
		}

		// Build map: placeholder → actual value from input
		// matchedValues[0] = full match
		// matchedValues[1:] = captured groups (placeholders)
		values := make(map[string]string, len(placeholderNames))
		for i, name := range placeholderNames {
			values[name] = matchedValues[i+1]
		}

		// Apply captured values to output template
		// Example:
		//   template: "Standard_D{size}_v3"
		//   values: {size: "4"}
		//   result: "Standard_D4_v3"
		resolved := replaceTemplatePlaceholders(outputTemplate, values)
		if resolved != "" {
			return resolved
		}
	}

	// If no templates matched, return input unchanged
	return machineType
}

// Matches placeholders like "{size}", "{c_size}", etc.
var placeholderRegex = regexp.MustCompile(`\{(\w+)\}`)

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

// templateToRegex converts a template into a regular expression
// and extracts placeholder names in order.
//
// Example:
//
//	"Standard_D{size}" →
//	  regex: "^Standard_D([a-zA-Z0-9]+)$"
//	  placeholders: ["size"]
func templateToRegex(template string) (*regexp.Regexp, []string) {
	var pattern strings.Builder
	pattern.WriteString("^")

	placeholderNames := make([]string, 0)
	lastIdx := 0

	// Iterate over all placeholder matches
	for _, match := range placeholderRegex.FindAllStringSubmatchIndex(template, -1) {
		fullStart, fullEnd := match[0], match[1]
		nameStart, nameEnd := match[2], match[3]

		// Add escaped literal part before the placeholder
		pattern.WriteString(regexp.QuoteMeta(template[lastIdx:fullStart]))

		// Extract placeholder name (e.g. "size")
		placeholderName := template[nameStart:nameEnd]
		placeholderNames = append(placeholderNames, placeholderName)

		// Placeholder values in current machine-type templates are alphanumeric.
		// This prevents matching already-versioned values such as:
		// Standard_D48s_v5 against Standard_D{size}
		pattern.WriteString(`([a-zA-Z0-9]+)`)

		lastIdx = fullEnd
	}

	// Add any remaining literal text after last placeholder
	pattern.WriteString(regexp.QuoteMeta(template[lastIdx:]))
	pattern.WriteString("$")

	return regexp.MustCompile(pattern.String()), placeholderNames
}

// replaceTemplatePlaceholders substitutes placeholders in a template
// with their resolved values.
//
// Example:
//
//	template: "Standard_D{size}_v3"
//	values: {size: "4"}
//	result: "Standard_D4_v3"
//
// If a placeholder has no value, it is left unchanged.
func replaceTemplatePlaceholders(template string, values map[string]string) string {
	return placeholderRegex.ReplaceAllStringFunc(template, func(token string) string {
		// Strip braces: "{size}" → "size"
		name := token[1 : len(token)-1]

		// Replace if value exists
		if value, ok := values[name]; ok {
			return value
		}

		// Otherwise keep original placeholder
		return token
	})
}
