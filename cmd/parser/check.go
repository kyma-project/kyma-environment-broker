package main

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(NewCheckCmd())
}

type CheckCommand struct {
	cobraCmd     *cobra.Command
	rule         string
	parser       rules.Parser
	ruleFilePath string
	match        string
	noColor      bool
}

func NewCheckCmd() *cobra.Command {
	cmd := CheckCommand{}
	cobraCmd := &cobra.Command{
		Use:     "check",
		Aliases: []string{"c"},
		Short:   "Check a HAP rules",
		Long:    "Check a HAP rules file validating its format and contents.",
		Example: `

	# Check multiple rules from a file:
	# --- rules.yaml
	# rule:
	# - azure(PR=westeurope)
	# - aws->EU 
	# ---
	hap check -f rules.yaml

	# Check multiple rules from a command line arguments
	hap check -e 'azure(PR=westeurope); aws->EU' 

	# Check which rule will be matched and triggered against the provided provisioning data
	hap check -f ./correct-rules.yaml -m '{"plan": "aws", "platformRegion": "cf-eu11", "hyperscalerRegion": "westeurope"}'
		`,
		RunE: func(_ *cobra.Command, args []string) error {
			cmd.Run()
			return nil
		},
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.rule, "entry", "e", "", "A rule to validate where each rule entry is separated by comma.")
	cobraCmd.Flags().StringVarP(&cmd.match, "match", "m", "", "Check which rule will be matched and triggered against the provided provisioning data. Data is passed in json format, example: '{\"plan\": \"aws\", \"platformRegion\": \"cf-eu11\"}'.")
	cobraCmd.Flags().StringVarP(&cmd.ruleFilePath, "file", "f", "", "Read rules from a yaml file. Rules are specified as a list of strings.")
	cobraCmd.Flags().BoolVarP(&cmd.noColor, "no-color", "n", false, "Disable coloring for output.")
	cobraCmd.MarkFlagsOneRequired("file", "entry")

	return cobraCmd
}

func (cmd *CheckCommand) Run() {

	printer := rules.NewColored(cmd.cobraCmd.Printf)
	if cmd.noColor {
		printer = rules.NewNoColor(cmd.cobraCmd.Printf)
	}

	// TODO: this method does not take current configuration into account and always returns all plans defined in the source file
	enabledPlans := broker.EnablePlans{}
	for _, plan := range broker.PlanNamesMapping {
		enabledPlans = append(enabledPlans, plan)
	}

	var rulesService *rules.RulesService
	var err error
	if cmd.ruleFilePath != "" {
		cmd.cobraCmd.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		//TODO: using stdin or file would require to change of NewRulesServiceFromFile method to accept io.Reader
		rulesService, err = rules.NewRulesServiceFromFile(cmd.ruleFilePath, &enabledPlans, true, true, true)
	} else {
		rulesService, err = rules.NewRulesServiceFromString(cmd.rule, &enabledPlans, true, true, true)
	}

	if err != nil {
		cmd.cobraCmd.Printf("Error: %s\n", err)
	}

	var dataForMatching *rules.ProvisioningAttributes
	if cmd.match != "" {
		dataForMatching = getDataForMatching(cmd.match)
	} else {
		dataForMatching = &rules.ProvisioningAttributes{
			PlatformRegion:    "<pr>",
			HyperscalerRegion: "<hr>",
		}
	}

	var matchingResults map[uuid.UUID]*rules.MatchingResult
	if cmd.match != "" && dataForMatching != nil {
		matchingResults = rulesService.Match(dataForMatching)
	}

	printer.Print(rulesService.Parsed.Results, matchingResults)

	hasErrors := false
	for _, result := range rulesService.Parsed.Results {
		if result.HasErrors() {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		cmd.cobraCmd.Printf("There are errors in your rule configuration. Fix above errors in your rule configuration and try again.\n")
	}
}

func getDataForMatching(content string) *rules.ProvisioningAttributes {
	data := &rules.ProvisioningAttributes{}
	err := json.Unmarshal([]byte(content), data)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return data
}
