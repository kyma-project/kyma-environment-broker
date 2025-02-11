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
    Case []TestCase `yaml:"cases"`
}

type TestCase struct {
	Rules []string `yaml:"rule"`
    ExpectedRule string `yaml:"expected"`
}

func (c *TestCases) loadCases() *TestCases {
	yamlFile, err := os.ReadFile("rules/test-cases.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}


func TestMain(t *testing.T) {
 
    t.Run("should parse simple plan", func (t *testing.T)  {

        cases := TestCases{}
        cases.loadCases()

        for _, c := range cases.Case {
        
            expected := strings.ReplaceAll(c.ExpectedRule, " ", "")
            expected = strings.ReplaceAll(expected, "\t", "")
            expected = strings.ReplaceAll(expected, "\n", "")
            expected = strings.ReplaceAll(expected, "\r", "")
            expected = strings.ReplaceAll(expected, "\f", "")
        
            entries := ""
            for i, rule := range c.Rules {
                entries += rule 
                
                if i < len(c.Rules) - 1 {
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

            output := strings.ReplaceAll(string(out), " ", "")
            output = strings.ReplaceAll(output, "\t", "")
            output = strings.ReplaceAll(output, "\n", "")
            output = strings.ReplaceAll(output, "\r", "")
            output = strings.ReplaceAll(output, "\f", "")

            fmt.Print("Output is: " + string(out))
            require.Equal(t, expected, strings.Trim(output, "\n"))
        }


    })
 
}