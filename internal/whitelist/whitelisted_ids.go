package whitelist

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

const (
	Key = "whitelist"
)

type Set map[string]struct{}

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
