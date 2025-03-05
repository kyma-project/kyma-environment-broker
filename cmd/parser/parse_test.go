package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type TestCases struct {
	Case []*TestCase `yaml:"cases"`
}

type TestCase struct {
	Name         string   `yaml:"name"`
	Rules        []string `yaml:"rule"`
	ExpectedRule string   `yaml:"expected"`
}

func (c *TestCases) loadCases() *TestCases {
	yamlFile, err := os.ReadFile("rules/test-cases.yaml")
	if err != nil {
		log.Printf("err while reading a file %v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (c *TestCases) writeCases() *TestCases {
	os.Remove("rules/test-cases.yaml")

	bytes, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}

	err = os.WriteFile("rules/test-cases.yaml", bytes, os.ModePerm)
	if err != nil {
		log.Printf("err while writing a file %v ", err)
	}

	return c
}

func TestMain(t *testing.T) {

	t.Run("should verify parser command", func(t *testing.T) {
		cases := TestCases{}
		cases.loadCases()

		overwrite := false

		for _, c := range cases.Case {
			log.Printf("Input:\n %s", c.Rules)
			log.Printf("Expected formatted:\n %s", c.ExpectedRule)
			expected := strings.ReplaceAll(c.ExpectedRule, " ", "")
			expected = strings.ReplaceAll(expected, "\t", "")
			expected = strings.ReplaceAll(expected, "\n", "")
			expected = strings.ReplaceAll(expected, "\r", "")
			expected = strings.ReplaceAll(expected, "\f", "")

			entries := ""
			for i, rule := range c.Rules {
				entries += rule

				if i < len(c.Rules)-1 {
					entries += "; "
				}
			}

			cmd := NewParseCmd()
			b := bytes.NewBufferString("")
			cmd.SetOut(b)

			cmd.SetArgs([]string{"-e", entries, "-nups"})
			cmd.Execute()
			out, err := io.ReadAll(b)
			if err != nil {
				t.Fatal(err)
			}

			if overwrite {
				c.ExpectedRule = string(out)
			} else {
				log.Printf("Actual formatted:\n %s", out)
				output := strings.ReplaceAll(string(out), " ", "")
				output = strings.ReplaceAll(output, "\t", "")
				output = strings.ReplaceAll(output, "\n", "")
				output = strings.ReplaceAll(output, "\r", "")
				output = strings.ReplaceAll(output, "\f", "")

				require.Equal(t, expected, strings.Trim(output, "\n"), fmt.Sprintf("While evaluating: %s", string(c.Name)))
			}

		}

		if overwrite {
			cases.writeCases()
		}
	})

}
