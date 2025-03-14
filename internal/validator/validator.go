package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type dlclarkRegexp regexp2.Regexp

func (re *dlclarkRegexp) MatchString(s string) bool {
	matched, err := (*regexp2.Regexp)(re).MatchString(s)
	return err == nil && matched
}

func (re *dlclarkRegexp) String() string {
	return (*regexp2.Regexp)(re).String()
}

func dlclarkCompile(s string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(s, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*dlclarkRegexp)(re), nil
}

func NewFromSchema(schema map[string]any) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft4)
	// The default Go regex engine does not support negative lookahead,
	// which is required for validating additional worker node pool names.
	compiler.UseRegexpEngine(dlclarkCompile)
	if err := compiler.AddResource("schema.json", schema); err != nil {
		return nil, fmt.Errorf("while adding schema: %w", err)
	}

	validator, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("while compiling schema: %w", err)
	}

	return validator, nil
}

func FormatError(err error) string {
	var validationError *jsonschema.ValidationError
	if errors.As(err, &validationError) {
		var errMsgs []string
		for _, cause := range validationError.Causes {
			errMsgs = append(errMsgs, cause.Error())
		}
		return strings.Join(errMsgs, ", ")
	}
	return "while formatting validation error"
}
