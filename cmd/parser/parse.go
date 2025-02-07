package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/grammar"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const colorError = "\033[0;31m"
const colorOk= "\033[32m" 
const colorNeutral = "\033[0m"
const colorMatched = "\033[34m"

func init() {
	rootCmd.AddCommand(NewParseCmd())
}

type ParseCommand struct {
	cobraCmd               *cobra.Command
	rule 				 string
	parser 				 rules.Parser
	useGrammar 			 bool
	ruleFilePath 		 string
	sort 			 bool
	unique 			 bool
	match 			 string
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
	hap parse -e 'azure(PR=westeurope), aws->EU' 
	
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
		`,

		RunE:    func(_ *cobra.Command, args []string) error { 
			return cmd.Run() 
		},
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.rule, "entry", "e", "", "A rule to validate where each rule entry is separated by comma.")
	cobraCmd.Flags().StringVarP(&cmd.match, "match", "m", "", "Check what rule will be matched and triggered against the provided test data. Only valid entries are taking into account when matching. Data is passed in json format, example: '{\"plan\": \"aws\", \"platformRegion\": \"cf-eu11\"}'.")
	cobraCmd.Flags().StringVarP(&cmd.ruleFilePath, "file", "f", "", "Read rules from a file pointed to by parameter value. The file must contain a valid yaml list, where each rule entry starts with '-' and is placed in its own line.")
	cobraCmd.Flags().BoolVarP(&cmd.useGrammar, "grammar", "g", false, "Use c parser and lexer generated with antlr instead of simple string splitting.")
	cobraCmd.Flags().BoolVarP(&cmd.sort, "priority", "p", false, "Sort rule entries by their priority, in descending priority order.")
	cobraCmd.Flags().BoolVarP(&cmd.unique, "unique", "u", false, "Display only non duplicated rules. Error entries are not considered for uniqueness.")
	cobraCmd.MarkFlagsOneRequired("entry", "file")

	return cobraCmd
}

func (cmd *ParseCommand) Run() error {
	cmd.parser = &rules.SimpleParser{}

	if cmd.match != "" && (!cmd.sort || !cmd.unique) {
		fmt.Printf("\tMatching is only supported when both priority and uniqnuess flags are specified.\n")
		return nil
	}
	
	if (cmd.useGrammar) {
		cmd.parser = &grammar.GrammarParser{}
	}

	var entries []string
	if cmd.ruleFilePath != "" {
		conf := &conf{}
		conf.getConf(cmd.ruleFilePath)
		fmt.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		entries = conf.Rules
	} else {
		entries = strings.Split(cmd.rule, ",")
	}

	allResults := make([]rules.ParsingResult, 0, len(entries))
	okResults := make([]rules.ParsingResult, 0, len(entries))
	errorResults := make([]rules.ParsingResult, 0, len(entries))
	for _, entry := range entries {
		rule, err := cmd.parser.Parse(entry)

		result := rules.ParsingResult{OriginalRule: entry,  Rule: rule, Err: err}
	
		if err != nil {
			errorResults = append(errorResults, result)	
		} else {
			okResults = append(okResults, result)	
		}

		allResults = append(allResults, result)
	}

	if cmd.sort {
		allResults = rules.SortRuleEntries(allResults)
		okResults = rules.SortRuleEntries(allResults)
		errorResults = rules.SortRuleEntries(allResults)
	}

	if cmd.unique {
		uniqnuessSet := make(map[string]rules.ParsingResult)

		uniqueResults := make([]rules.ParsingResult, 0, len(allResults))
		for _, result := range allResults {

			if result.Err != nil {
				uniqueResults = append(uniqueResults, result)
				continue
			}

			if item, ok := uniqnuessSet[result.Rule.Plan + result.Rule.PlatformRegion + result.Rule.HyperscalerRegion]; !ok {
				uniqnuessSet[result.Rule.Plan + result.Rule.PlatformRegion + result.Rule.HyperscalerRegion] = result
				uniqueResults = append(uniqueResults, result)
			} else {
				errorResults = append(errorResults, rules.ParsingResult{OriginalRule: result.OriginalRule,  Rule: result.Rule, Err: fmt.Errorf("Duplicated rule with previously defined rule: %s", item.OriginalRule)})
			}
		}

		allResults = uniqueResults
	}


	var testDataForMatching *rules.MatchableAttributes
	if (cmd.match != "") {
		testDataForMatching = getTestData(cmd.match)
	}
	
	for _, result := range allResults {

		if result.Err != nil {
			fmt.Printf("\t\t-> %s Error %s parsing rule: %s\n", colorError, colorNeutral, result.Err)
		} else {

			fmt.Printf("\t\t-> ")

			if (cmd.match != "" && testDataForMatching != nil) {
				matched := result.Rule.Matched(testDataForMatching)
				if matched {
					fmt.Printf("%s Matched %s - ", colorMatched, colorNeutral)
				} 
			}

			fmt.Printf("%s OK %s %s\n", colorOk, colorNeutral, result.Rule.String())
		}
	}

	if len(errorResults) != 0 {
		fmt.Printf("\tThere are errors in your rule configuration. Fix above errors in your rule configuration and try again.\n")
		return nil
	}



	return nil
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

