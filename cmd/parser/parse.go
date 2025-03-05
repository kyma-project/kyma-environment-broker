package main

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
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
		Short:   "Parses a HAP rule entry validating its format",
		Long:    "Parses a HAP rule entry validating its format, by default using simple string splitting. Documentation can be found ... .",
		Example: `
	# Parse a rule entry using simple string splitting
	hap parse -e 'azure(PR=westeurope), aws->EU' 
	
	# Parse multiple rules from a file using simple string splitting
	hap parse -e 'azure(PR=westeurope); aws->EU' 

	# Parse multiple rules from a file:
	# --- rules.yaml
	# rule:
	# - azure(PR=westeurope)
	# - aws->EU 
	# ---
	hap parse -f rules.yaml

	# Sort rule entries by their priority
	hap parse -p -e 'azure(PR=westeurope), aws->EU'	
	
	# Disable duplicated rule entries
	hap parse -u -e 'azure(PR=westeurope), azure(PR=westeurope)'

	# Check what rule will be matched and triggered against the provided test data
	hap parse -p -u  -f ./correct-rules.yaml -m '{"plan": "aws", "platformRegion": "cf-eu11", "hyperscalerRegion": "westeurope"}'
		`,

		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run()
		},
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.rule, "entry", "e", "", "A rule to validate where each rule entry is separated by comma.")
	cobraCmd.Flags().StringVarP(&cmd.match, "match", "m", "", "Check what rule will be matched and triggered against the provided test data. Only valid entries are taking into account when matching. Data is passed in json format, example: '{\"plan\": \"aws\", \"platformRegion\": \"cf-eu11\"}'.")
	cobraCmd.Flags().StringVarP(&cmd.ruleFilePath, "file", "f", "", "Read rules from a file pointed to by parameter value. The file must contain a valid yaml list, where each rule entry starts with '-' and is placed in its own line.")
	cobraCmd.Flags().BoolVarP(&cmd.sort, "priority", "p", false, "Sort rule entries by their priority, in descending priority order.")
	cobraCmd.Flags().BoolVarP(&cmd.unique, "unique", "u", false, "Display only non duplicated rules. Error entries are not considered for uniqueness.")
	cobraCmd.Flags().BoolVarP(&cmd.signature, "signature", "s", false, "Mark rules with the mirrored signatures as duplicated. For example aws(PR=*, HR=westeurope) and aws(PR=westeurope, HR=*) are considered duplicated because of having mirrored signatures.")
	cobraCmd.Flags().BoolVarP(&cmd.noColor, "no-color", "n", false, "Disable use color characters when generating output.")
	cobraCmd.MarkFlagsOneRequired("entry", "file")

	return cobraCmd
}

type ProcessingPair struct {
	ParsingResults  *rules.ParsingResult
	MatchingResults *rules.MatchingResult
}

func (cmd *ParseCommand) Run() error {

	printer := rules.NewColored(cmd.cobraCmd.Printf)
	if cmd.noColor {
		printer = rules.NewNoColor(cmd.cobraCmd.Printf)
	}

	if cmd.match != "" && (!cmd.sort || !cmd.unique) {
		cmd.cobraCmd.Printf("\tMatching is only supported when both priority and uniqueness flags are specified.\n")
		return nil
	}

	var rulesService *rules.RulesService
	var err error
	if cmd.ruleFilePath != "" {
		cmd.cobraCmd.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		rulesService, err = rules.NewRulesServiceFromFile(cmd.ruleFilePath, cmd.sort, cmd.unique, cmd.signature)
	} else {
		rulesService, err = rules.NewRulesServiceFromString(cmd.rule, cmd.sort, cmd.unique, cmd.signature)
	}

	if err != nil {
		cmd.cobraCmd.Printf("Error: %s\n", err)
		return nil
	}

	var dataForMatching *rules.MatchableAttributes
	if cmd.match != "" {
		dataForMatching = getDataForMatching(cmd.match)
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
		return nil
	}

	return nil
}

type conf struct {
	Rules []string `yaml:"rule"`
}

func getDataForMatching(content string) *rules.MatchableAttributes {
	testData := &rules.MatchableAttributes{}
	err := json.Unmarshal([]byte(content), testData)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return testData
}
