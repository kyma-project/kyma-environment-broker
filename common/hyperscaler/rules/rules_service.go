package rules

import (
	"fmt"
	"log"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/model"
)

type RulesService struct {
}

func NewRulesServiceFromFile(rulesFilePath string) (*RulesService, error) {
	rulesConfig := &model.RulesConfig{}

	if rulesFilePath == "" {
		return nil, fmt.Errorf("No HAP rules file path provided")
	}

	log.Printf("Parsing rules from file: %s\n", rulesFilePath)
	err := rulesConfig.Load(rulesFilePath)
	if err != nil {
		return nil, err
	}

	return &RulesService{}, nil
}
