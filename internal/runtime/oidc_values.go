package runtime

import (
	"fmt"
	"os"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"gopkg.in/yaml.v3"
)

func ReadOIDCDefaultValuesFromYAML(yamlFilePath string) (pkg.OIDCConfigDTO, error) {
	var values pkg.OIDCConfigDTO
	yamlFile, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return pkg.OIDCConfigDTO{}, fmt.Errorf("while reading YAML file with OIDC default values: %w", err)
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return pkg.OIDCConfigDTO{}, fmt.Errorf("while unmarshalling YAML file with OIDC default values: %w", err)
	}
	return values, nil
}
