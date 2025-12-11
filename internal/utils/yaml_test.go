package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalingFromYamlFile(t *testing.T) {

	tests := []struct {
		name     string
		filename string
		expected interface{}
		data     interface{}
	}{
		{
			name:     "correctly unmarshals yaml file with list of strings",
			filename: "testdata/list_of_string_in_property.yaml",
			data:     &map[string][]string{},
			expected: &map[string][]string{"prop": {"string-1", "string-2"}},
		},
		{
			name:     "correctly unmarshals yaml file with embedded object",
			filename: "testdata/embedded_object.yaml",
			data:     &map[string]interface{}{},
			expected: &map[string]interface{}{"obj": map[string]interface{}{"prop1": map[string]interface{}{"another_obj": map[string]interface{}{"prop2": "value2", "prop3": "value3"}}}},
		},
		{
			name:     "correctly unmarshals yaml file with mapping of strings to list of strings",
			filename: "testdata/multiple_mappings.yaml",
			data:     &map[string][]string{},
			expected: &map[string][]string{"key1": {"value1", "value2"}, "key2": {"value3", "value4"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given/when
			err := UnmarshalYamlFile(tt.filename, tt.data)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tt.data)
		})
	}
}
