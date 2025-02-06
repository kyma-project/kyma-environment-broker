package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/grammar"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

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
}

func NewParseCmd() *cobra.Command {
	cmd := ParseCommand{}
	cobraCmd := &cobra.Command{
		Use:     "parse",
		Aliases: []string{"p"},
		Short:   "Parses a HAP rule entry validating its format",
		Long:    "Parses a HAP rule entry validating its format, by default using simple string splitting. Documentation can be found ... . ",
		Example: `parser parse 'azure'`,

		RunE:    func(_ *cobra.Command, args []string) error { 
			return cmd.Run() 
		},
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.rule, "entry", "e", "", "A rule to validate where each rule entry is separated by comma.")
	cobraCmd.Flags().StringVarP(&cmd.ruleFilePath, "file", "f", "", "Read rules from a file pointed to by parameter value. The file must contain a valid yaml list, where each rule entry starts with '-' and is placed in its own line.")
	cobraCmd.Flags().BoolVarP(&cmd.useGrammar, "grammar", "g", false, "Use c parser and lexer generated with antlr instead of simple string splitting.")
	cobraCmd.Flags().BoolVarP(&cmd.sort, "sort", "s", false, "Sort rule entries by their priority.")
	cobraCmd.MarkFlagsOneRequired("entry", "file")

	return cobraCmd
}

func (cmd *ParseCommand) Run() error {
	cmd.parser = &rules.SimpleParser{}
	
	if (cmd.useGrammar) {
		cmd.parser = &grammar.GrammarParser{}
	}

	var entries []string
	if cmd.ruleFilePath != "" {
		conf := &conf{}
		conf.getConf()
		fmt.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		entries = conf.Rules
	} else {
		entries = strings.Split(cmd.rule, ",")
	}

	results := make([]rules.ParsingResult, 0, len(entries))
	for _, entry := range entries {
		rule, err := cmd.parser.Parse(entry)
	
		results = append(results, rules.ParsingResult{OriginalRule: entry,  Rule: rule, Err: err})	
	}

	if cmd.sort {
		results = rules.SortRuleEntries(results)
	}

	for _, result := range results {
		fmt.Printf("Parsing rule: %s\n", result.OriginalRule)

		if result.Err != nil {
			fmt.Printf("-> Error parsing rule: %s\n", result.Err)
		} else {
			fmt.Printf("-> Parsed rule: %+v\n", result.Rule)
		}
	}

	return nil
}

type conf struct {
	Rules []string `yaml:"rule"`
}

func (c *conf) getConf() *conf {
	yamlFile, err := os.ReadFile("resources/rules.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

