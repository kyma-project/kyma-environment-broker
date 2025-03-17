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
	rootCmd.AddCommand(NewParseCmd())
}

type ParseCommand struct {
	cobraCmd     *cobra.Command
	rule         string
	parser       rules.Parser
	ruleFilePath string
	sort         bool
	unique       bool
	match        string
	signature    bool
	noColor      bool
}

func NewParseCmd() *cobra.Command {
	cmd := ParseCommand{}
	cobraCmd := &cobra.Command{
		Use:     "parse",
		Aliases: []string{"p"},
		Short:   "Parses a HAP rule entry validating its format.",
		Long:    "Parses a HAP rule entry validating its format.",
		Example: `
	# Parse multiple rules from command line arguments
	hap parse -e 'azure(PR=westeurope); aws->EU' 

	# Parse multiple rules from a file:
	# --- rules.yaml
	# rule:
	# - azure(PR=westeurope)
	# - aws->EU 
	# ---
	hap parse -f rules.yaml

	# Check which rule will be matched and triggered against the provided provisioning data
	hap parse  -f ./correct-rules.yaml -m '{"plan": "aws", "platformRegion": "cf-eu11", "hyperscalerRegion": "westeurope"}'
		`,
		RunE: func(_ *cobra.Command, args []string) error {
			cmd.Run()
			return nil
		},
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.rule, "entry", "e", "", "A rule to validate where each rule entry is separated by comma.")
	cobraCmd.Flags().StringVarP(&cmd.match, "match", "m", "", "Check what rule will be matched and triggered against the provided test data. Only valid entries are taking into account when matching. Data is passed in json format, example: '{\"plan\": \"aws\", \"platformRegion\": \"cf-eu11\"}'.")
	cobraCmd.Flags().StringVarP(&cmd.ruleFilePath, "file", "f", "", "Read rules from a file pointed to by parameter value. The file must contain a valid yaml list, where each rule entry starts with '-' and is placed in its own line.")
	cobraCmd.Flags().BoolVarP(&cmd.noColor, "no-color", "n", false, "Disable use color characters when generating output.")
	cobraCmd.MarkFlagsOneRequired("entry", "file")

	return cobraCmd
}

type ProcessingPair struct {
	ParsingResults  *rules.ParsingResult
	MatchingResults *rules.MatchingResult
}

func (cmd *ParseCommand) Run() {

	printer := rules.NewColored(cmd.cobraCmd.Printf)
	if cmd.noColor {
		printer = rules.NewNoColor(cmd.cobraCmd.Printf)
	}

	// create enabled plans
	enabledPlans := broker.EnablePlans{}
	for _, plan := range broker.PlanNamesMapping {
		enabledPlans = append(enabledPlans, plan)
	}

	var rulesService *rules.RulesService
	var err error
	if cmd.ruleFilePath != "" {
		cmd.cobraCmd.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		rulesService, err = rules.NewRulesServiceFromFile(cmd.ruleFilePath, &enabledPlans)
	} else {
		rulesService, err = rules.NewRulesServiceFromString(cmd.rule, &enabledPlans)
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

type conf struct {
	Rules []string `yaml:"rule"`
}

func getDataForMatching(content string) *rules.ProvisioningAttributes {
	data := &rules.ProvisioningAttributes{}
	err := json.Unmarshal([]byte(content), data)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return data
}
