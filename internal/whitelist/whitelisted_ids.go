package whitelist

import (
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

const (
	Key = "whitelist"
)

type Set map[string]struct{}

func (s Set) Contains(id string) bool {
	return s != nil && IsWhitelisted(id, s)
}

func (s Set) String() string {
	if len(s) > 20 {
		return fmt.Sprintf("[Set with %d elements]", len(s))
	}
	// join all keys by comma
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return fmt.Sprintf("[%s]", strings.Join(keys, ", "))
}

func IsWhitelisted(id string, whitelist Set) bool {
	_, found := whitelist[id]
	return found
}

func IsNotWhitelisted(id string, whitelist Set) bool {
	_, found := whitelist[id]
	return !found
}

func ReadWhitelistedIdsFromFile(filename string) (Set, error) {
	yamlData := make(map[string][]string)
	err := utils.UnmarshalYamlFile(filename, &yamlData)
	if err != nil {
		return Set{}, fmt.Errorf("while unmarshalling a file with whitelisted ids config: %w", err)
	}

	whitelistSet := Set{}
	for _, id := range yamlData[Key] {
		whitelistSet[id] = struct{}{}
	}
	return whitelistSet, nil
}
