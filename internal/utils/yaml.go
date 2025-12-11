package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func UnmarshalYamlFile(filename string, out interface{}) error {
	var fileBytes, err = os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("while reading a %s file : %w", filename, err)
	}

	// Parse the YAML into nodes to handle duplicate keys manually
	var node yaml.Node
	if err := yaml.Unmarshal(fileBytes, &node); err != nil {
		return fmt.Errorf("while unmarshaling yaml data: %w", err)
	}

	// Convert node to the appropriate format with yaml.v2 compatibility
	result := convertNode(&node)

	// Assign to output based on type
	switch v := out.(type) {
	case *map[string]interface{}:
		if m, ok := result.(map[string]interface{}); ok {
			*v = m
		} else if m2, ok := result.(map[interface{}]interface{}); ok {
			// Convert map[interface{}]interface{} to map[string]interface{}
			converted := make(map[string]interface{})
			for key, val := range m2 {
				if strKey, ok := key.(string); ok {
					converted[strKey] = val
				}
			}
			*v = converted
		} else {
			return fmt.Errorf("unexpected result type: %T", result)
		}
	case *map[string][]string:
		var m map[string]interface{}
		if mi, ok := result.(map[interface{}]interface{}); ok {
			m = make(map[string]interface{})
			for key, val := range mi {
				if strKey, ok := key.(string); ok {
					m[strKey] = val
				}
			}
		} else if ms, ok := result.(map[string]interface{}); ok {
			m = ms
		}
		if m != nil {
			converted := make(map[string][]string)
			for key, val := range m {
				if arr, ok := val.([]interface{}); ok {
					strArr := make([]string, len(arr))
					for i, item := range arr {
						if str, ok := item.(string); ok {
							strArr[i] = str
						}
					}
					converted[key] = strArr
				}
			}
			*v = converted
		}
	default:
		return fmt.Errorf("unsupported output type: %T", out)
	}

	return nil
}

// convertNode processes yaml.Node and handles duplicate keys (last wins) with yaml.v2 compatibility
func convertNode(node *yaml.Node) interface{} {
	return convertNodeInternal(node, true)
}

func convertNodeInternal(node *yaml.Node, isTopLevel bool) interface{} {
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return convertNodeInternal(node.Content[0], isTopLevel)
		}
		return nil
	case yaml.MappingNode:
		// Nested objects use map[interface{}]interface{} (yaml.v2 behavior)
		// Top-level uses map[string]interface{} to match pointer type
		if !isTopLevel {
			m := make(map[interface{}]interface{})
			for i := 0; i < len(node.Content); i += 2 {
				keyNode := node.Content[i]
				valueNode := node.Content[i+1]

				// Decode key as string
				var key string
				keyNode.Decode(&key)

				value := convertNodeInternal(valueNode, false)
				m[key] = value // Last duplicate key wins (yaml.v2 behavior)
			}
			return m
		} else {
			// Top level: use map[string]interface{}
			m := make(map[string]interface{})
			for i := 0; i < len(node.Content); i += 2 {
				keyNode := node.Content[i]
				valueNode := node.Content[i+1]

				var key string
				keyNode.Decode(&key)

				value := convertNodeInternal(valueNode, false)
				m[key] = value
			}
			return m
		}

	case yaml.SequenceNode:
		var s []interface{}
		for _, item := range node.Content {
			s = append(s, convertNodeInternal(item, false))
		}
		return s
	case yaml.ScalarNode:
		var value interface{}
		node.Decode(&value)
		return value
	case yaml.AliasNode:
		return convertNodeInternal(node.Alias, false)
	default:
		return nil
	}
}
