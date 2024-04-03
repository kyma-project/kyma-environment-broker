package whitelist

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	Key = "whitelist"
)

type Set map[string]struct{}

func IsNotWhitelisted(globalAccountId string, whitelist Set) bool {
	_, found := whitelist[globalAccountId]
	return !found
}

func ReadWhitelistedGlobalAccountIdsFromFile(filename string) (Set, error) {
	yamlData := make(map[string][]string)
	whitelistSet := Set{}
	var whitelist, err = os.ReadFile(filename)
	if err != nil {
		return whitelistSet, fmt.Errorf("while reading %s file with whitelisted GlobalAccountIds config: %w", filename, err)
	}
	err = yaml.Unmarshal(whitelist, &yamlData)
	if err != nil {
		return whitelistSet, fmt.Errorf("while unmarshalling a file with whitelisted GlobalAccountIds config: %w", err)
	}
	for _, id := range yamlData[Key] {
		whitelistSet[id] = struct{}{}
	}
	return whitelistSet, nil
}
