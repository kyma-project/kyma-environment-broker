package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var colorError = "\033[0;31m"
var colorOk= "\033[32m" 
var colorNeutral = "\033[0m"
var colorMatched = "\033[34m"

func init() {
	rootCmd.AddCommand(NewParseCmd())
}

type ParseCommand struct {
	cobraCmd               *cobra.Command
	rule 				 string
	parser 				 rules.Parser
	ruleFilePath 		 string
	sort 			 bool
	unique 			 bool
	match 			 string
	signature 			 bool
	noColor 			 bool
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
	
	# Parse a rule entry using antlr parser and lexer
	hap parse -g -e 'azure(PR=westeurope)' 

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
	hap parse -p -u  -f ./correct-rules.yaml -m '{"plan": "aws"}'
		`,

		RunE:    func(_ *cobra.Command, args []string) error { 
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

func (cmd *ParseCommand) Run() error {
	cmd.parser = &rules.SimpleParser{}

	if cmd.noColor {
		colorError = ""
		colorOk = ""
		colorNeutral = ""
		colorMatched = ""
	}

	if cmd.match != "" && (!cmd.sort || !cmd.unique) {
		cmd.cobraCmd.Printf("\tMatching is only supported when both priority and uniqueness flags are specified.\n")
		return nil
	}
	
	var entries []string
	if cmd.ruleFilePath != "" {
		conf := &conf{}
		conf.getConf(cmd.ruleFilePath)
		cmd.cobraCmd.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		entries = conf.Rules
	} else {
		entries = strings.Split(cmd.rule, ";")
	}

	results := rules.NewParsingResults()

	for _, entry := range entries {
		rule, err := cmd.parser.Parse(entry)

		results.Apply(entry, rule, err)
	}

	if cmd.sort {
		results.Sort()
	}

	if cmd.unique {
		results.CheckUniqueness()
	}

	if cmd.signature {
		results.CheckSignatures()
	}

	if cmd.sort {
		results.Sort()
	}

	var testDataForMatching *rules.MatchableAttributes
	if (cmd.match != "") {
		testDataForMatching = getTestData(cmd.match)
	}
	
	Print(cmd, results, testDataForMatching)
	

	if results.HasErrors(){
		cmd.cobraCmd.Printf("There are errors in your rule configuration. Fix above errors in your rule configuration and try again.\n")
		return nil
	}

	return nil
}

func Print(cmd *ParseCommand, results *rules.ParsingResults, testDataForMatching *rules.MatchableAttributes) {


	if cmd.match != ""  && testDataForMatching != nil {
		var lastMatch *rules.ParsingResult = nil
		for _, result := range results.AllResults {
			if result.Err == nil {
				result.Matched = result.Rule.Matched(testDataForMatching)
				if result.Matched {
					lastMatch = result
				}
			}
		}

		if lastMatch != nil {
			lastMatch.FinalMatch = true
		}
	}

	for _, result := range results.AllResults {

		cmd.cobraCmd.Printf("-> ")
		if result.Err != nil {

			cmd.cobraCmd.Printf("%s Error %s", colorError, colorNeutral)

		} else {


			cmd.cobraCmd.Printf("%s %5s %s", colorOk, "OK", colorNeutral)
		}

		if result.Rule != nil && result.Err == nil {
			cmd.cobraCmd.Printf(" %s", result.Rule.String())
		}

		if result.Err != nil {
			cmd.cobraCmd.Printf(" %s", result.OriginalRule)
			cmd.cobraCmd.Printf(" - %s", result.Err)
		}

		if (result.Err == nil && cmd.match != "" && testDataForMatching != nil) {
			if result.Matched && !result.FinalMatch {
				cmd.cobraCmd.Printf("%s Matched %s ", colorMatched, colorNeutral)
			} else if result.FinalMatch {
				cmd.cobraCmd.Printf("%s Matched, Selected %s ", colorMatched, colorNeutral)
			}
		}

		cmd.cobraCmd.Printf("\n")
	}
}

func resolvingSignatureFormat(item rules.ParsingResult) string {
	positiveSignature := item.Rule.Plan
	if item.Rule.PlatformRegion == "*" || item.Rule.PlatformRegion != "" {
		positiveSignature += "PR:attr"
	}

	if item.Rule.HyperscalerRegion == "*" || item.Rule.HyperscalerRegion != "" {
		positiveSignature += "HR:attr"
	}
	return positiveSignature
}

func resolvingSignature(item1, item2 rules.ParsingResult) string{
	resolvingSignature := item1.Rule.Plan

	for _, attribute := range rules.InputAttributes {
		if attribute.HasValue {
			var valueRule *rules.Rule

			if attribute.HasLiteral(item1.Rule) {
				valueRule = item1.Rule
			} else if attribute.HasLiteral(item2.Rule) {
				valueRule = item2.Rule
			} else {
				continue
			}

			resolvingSignature += attribute.Name + "=" + attribute.Getter(valueRule)
		}
	}
	
	// if item1.Rule.PlatformRegion != "*" && item1.Rule.PlatformRegion != "" {
	// 	resolvingSignature += "(PR=" + item1.Rule.PlatformRegion
	// } else if item2.Rule.PlatformRegion != "*" && item2.Rule.PlatformRegion != "" {
	// 	resolvingSignature += "(PR=" + item2.Rule.PlatformRegion
	// }

	// if item1.Rule.HyperscalerRegion != "*" && item1.Rule.HyperscalerRegion != "" {
	// 	resolvingSignature += "HR=" + item1.Rule.HyperscalerRegion
	// } else if item2.Rule.HyperscalerRegion != "*" && item2.Rule.HyperscalerRegion != "" {	
	// 	resolvingSignature += "HR=" + item2.Rule.HyperscalerRegion
	// }

	return resolvingSignature
}

type conf struct {
	Rules []string `yaml:"rule"`
}

func (c *conf) getConf(file string) *conf {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func getTestData(content string) *rules.MatchableAttributes {
	testData := &rules.MatchableAttributes{}
	err := json.Unmarshal([]byte(content), testData)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return testData
}

