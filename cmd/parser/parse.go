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
	signature 			 bool
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
	cobraCmd.Flags().BoolVarP(&cmd.useGrammar, "grammar", "g", false, "Use c parser and lexer generated with antlr instead of simple string splitting.")
	cobraCmd.Flags().BoolVarP(&cmd.sort, "priority", "p", false, "Sort rule entries by their priority, in descending priority order.")
	cobraCmd.Flags().BoolVarP(&cmd.unique, "unique", "u", false, "Display only non duplicated rules. Error entries are not considered for uniqueness.")
	cobraCmd.Flags().BoolVarP(&cmd.signature, "signature", "s", false, "Mark rules with the mirrored signatures as duplicated. For example aws(PR=*, HR=westeurope) and aws(PR=westeurope, HR=*) are considered duplicated because of having mirrored signatures.")
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

	resolvedRules := make(map[string]rules.ParsingResult)

	for _, entry := range entries {
		rule, err := cmd.parser.Parse(entry)

		result := rules.ParsingResult{OriginalRule: entry,  Rule: rule, Err: err}
	
		if err != nil {
			errorResults = append(errorResults, result)	
		} else {
			okResults = append(okResults, result)	
			if rule.IsResolved() {
				resolvedRules[rule.StringNoLabels()] = result
			}
		}

		allResults = append(allResults, result)
	}

	if cmd.sort {
		allResults = rules.SortRuleEntries(allResults)
		okResults = rules.SortRuleEntries(okResults)
		errorResults = rules.SortRuleEntries(errorResults)
	}

	if cmd.unique {
		uniqnuessSet := make(map[string]rules.ParsingResult)
		signatureSet := make(map[string]rules.ParsingResult)
		uniqueResults := make([]rules.ParsingResult, 0, len(allResults))


		for _, result := range allResults {

			containsWildcards := false

			if result.Err != nil {
				uniqueResults = append(uniqueResults, result)
				continue
			}

			negativeSignatureKey := result.Rule.Plan
			signatureKey := result.Rule.Plan
			key := result.Rule.Plan
			
			key += "PR:" 
			signatureKey += "PR:"
			negativeSignatureKey += "PR:"
			if result.Rule.PlatformRegion == "" || result.Rule.PlatformRegion == "*" {
				key += "*"

				signatureKey += "*"
				negativeSignatureKey += "attr"
				if result.Rule.PlatformRegion == "*" {
					containsWildcards = true
				}
			} else {
				key += result.Rule.PlatformRegion
			
				signatureKey += "attr"
				negativeSignatureKey += "*"
			}
			
			key += "HR:"
			signatureKey += "HR:"
			negativeSignatureKey += "HR:"
			if result.Rule.HyperscalerRegion == "" || result.Rule.HyperscalerRegion == "*" {
				key += "*"
			
				signatureKey += "*"
				negativeSignatureKey += "attr"

			
				if result.Rule.HyperscalerRegion == "*" {
					containsWildcards = true
				}

			} else {
				key += result.Rule.HyperscalerRegion
			
				signatureKey += "attr"
				negativeSignatureKey += "*"
			}

			negativeSignatureItem, negativeSignatureExists := signatureSet[negativeSignatureKey]

			

			if negativeSignatureExists && containsWildcards && cmd.signature {
				
				resolvingSignaturePossibleRule := result.Rule.Combine(*negativeSignatureItem.Rule)

			resolvingKey := resolvingSignaturePossibleRule.StringNoLabels()
			_, resolvingRuleExists := resolvedRules[resolvingKey]

				if !resolvingRuleExists {
					err := fmt.Errorf("Duplicated negative signature with previously defined rule: '%s', consider introducing a resolving rule '%s'", negativeSignatureItem.Rule.StringNoLabels(), resolvingKey)

					errorResults = append(errorResults, rules.ParsingResult{OriginalRule: result.OriginalRule, Err: err})

					uniqueResults = append(uniqueResults, rules.ParsingResult{OriginalRule: result.OriginalRule, Err: err})
				}
				continue
			}

			alreadyExists := false
			var item rules.ParsingResult
			item, alreadyExists = uniqnuessSet[key]
	
			if !alreadyExists {

				uniqnuessSet[key] = result
				signatureSet[signatureKey] = result
				uniqueResults = append(uniqueResults, result)

			} else {
				
				err := fmt.Errorf("Duplicated rule with previously defined rule: '%s'", item.Rule.StringNoLabels())

				errorResults = append(errorResults, rules.ParsingResult{OriginalRule: result.OriginalRule, Err: err})

				uniqueResults = append(uniqueResults, rules.ParsingResult{OriginalRule: result.OriginalRule, Err: err})

			}

		}

		allResults = uniqueResults
	}

	if cmd.sort {
		allResults = rules.SortRuleEntries(allResults)
		okResults = rules.SortRuleEntries(okResults)
		errorResults = rules.SortRuleEntries(errorResults)
	}

	var testDataForMatching *rules.MatchableAttributes
	if (cmd.match != "") {
		testDataForMatching = getTestData(cmd.match)
	}
	
	for _, result := range allResults {

		fmt.Printf("-> ")
		if result.Err != nil {

			fmt.Printf("%s Error %s", colorError, colorNeutral)

		} else {

			if (cmd.match != "" && testDataForMatching != nil) {
				matched := result.Rule.Matched(testDataForMatching)

				if matched {
					fmt.Printf("%s Matched %s ", colorMatched, colorNeutral)
				} 
			}

			fmt.Printf("%s %5s %s", colorOk, "OK", colorNeutral)
		}

		if result.Rule != nil && result.Err == nil {
			fmt.Printf(" %s", result.Rule.String())
		}

		if result.Err != nil {
			fmt.Printf(" %s", result.OriginalRule)
			fmt.Printf(" - %s", result.Err)
		}

		fmt.Printf("\n")
	}

	if len(errorResults) != 0 {
		fmt.Printf("There are errors in your rule configuration. Fix above errors in your rule configuration and try again.\n")
		return nil
	}

	return nil
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
	if item1.Rule.PlatformRegion != "*" && item1.Rule.PlatformRegion != "" {
		resolvingSignature += "(PR=" + item1.Rule.PlatformRegion
	} else if item2.Rule.PlatformRegion != "*" && item2.Rule.PlatformRegion != "" {
		resolvingSignature += "(PR=" + item2.Rule.PlatformRegion
	}

	if item1.Rule.HyperscalerRegion != "*" && item1.Rule.HyperscalerRegion != "" {
		resolvingSignature += "HR=" + item1.Rule.HyperscalerRegion
	} else if item2.Rule.HyperscalerRegion != "*" && item2.Rule.HyperscalerRegion != "" {	
		resolvingSignature += "HR=" + item2.Rule.HyperscalerRegion
	}

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

