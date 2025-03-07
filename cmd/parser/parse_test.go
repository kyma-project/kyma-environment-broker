package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const RULES_TEST_CASES = "rules/test-cases.yaml"

type TestCases struct {
	Case []*TestCase `yaml:"cases"`
}

type TestCase struct {
	Name         string   `yaml:"name"`
	Rules        []string `yaml:"rule"`
	ExpectedRule string   `yaml:"expected"`
}

func (c *TestCases) loadCases() {
	yamlFile, err := os.ReadFile(RULES_TEST_CASES)
	if err != nil {
		log.Printf("err while reading a file %v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

func (c *TestCases) writeCases() {
	os.Remove(RULES_TEST_CASES)

	bytes, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}

	err = os.WriteFile(RULES_TEST_CASES, bytes, os.ModePerm)
	if err != nil {
		log.Printf("err while writing a file %v ", err)
	}
}

func TestMain(t *testing.T) {

	t.Run("should verify parser command", func(t *testing.T) {
		cases := TestCases{}
		cases.loadCases()

		overwrite := false

		for _, c := range cases.Case {
			log.Printf("Input:\n %s", c.Rules)
			log.Printf("Expected formatted:\n %s", c.ExpectedRule)
			expected := rules.RemoveWhitespaces(c.ExpectedRule)

			entries := strings.Join(c.Rules, "; ")

			cmd := NewParseCmd()
			b := bytes.NewBufferString("")
			cmd.SetOut(b)

			cmd.SetArgs([]string{"-e", entries, "-nups"})
			err := cmd.Execute()
			require.NoError(t, err)

			out, err := io.ReadAll(b)
			if err != nil {
				t.Fatal(err)
			}

			if overwrite {
				c.ExpectedRule = string(out)
			} else {
				log.Printf("Actual formatted:\n %s", out)
				output := rules.RemoveWhitespaces(string(out))

				require.Equal(t, expected, strings.Trim(output, "\n"), fmt.Sprintf("While evaluating: %s", string(c.Name)))
			}

		}

		if overwrite {
			cases.writeCases()
		}
	})
}
