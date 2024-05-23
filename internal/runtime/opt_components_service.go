package runtime

import (
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

// ComponentDisabler disables component form the given list and returns modified list
type ComponentDisabler interface {
	Disable(components internal.ComponentConfigurationInputList) internal.ComponentConfigurationInputList
}

// ComponentsDisablers represents type for defining components disabler list
type ComponentsDisablers map[string]ComponentDisabler

// OptionalComponentsService provides functionality for executing component disablers
type OptionalComponentsService struct {
	registered map[string]ComponentDisabler
}

// NewOptionalComponentsService returns new instance of ResourceSupervisorAggregator
func NewOptionalComponentsService(initialList ComponentsDisablers) *OptionalComponentsService {
	return &OptionalComponentsService{
		registered: initialList,
	}
}

// GetAllOptionalComponentsNames returns list of registered components disablers names
func (f *OptionalComponentsService) GetAllOptionalComponentsNames() []string {
	var names []string
	for name := range f.registered {
		names = append(names, name)
	}

	return names
}

// AddComponentToDisable adds a component to the list of registered components disablers names
func (f *OptionalComponentsService) AddComponentToDisable(name string, disabler ComponentDisabler) {
	f.registered[name] = disabler
}

func toNormalizedMap(in []string) map[string]struct{} {
	out := map[string]struct{}{}

	for _, entry := range in {
		out[strings.ToLower(entry)] = struct{}{}
	}

	return out
}
