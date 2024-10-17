package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatedBy(t *testing.T) {
	emptyStr := ""
	email := "john.smith@email.com"
	origin := "origin"
	tests := []struct {
		name     string
		context  BindingContext
		expected string
	}{
		{
			name:     "Both Email and Origin are nil",
			context:  BindingContext{Email: nil, Origin: nil},
			expected: "",
		},
		{
			name:     "Both Email and Origin are empty",
			context:  BindingContext{Email: &emptyStr, Origin: &emptyStr},
			expected: "",
		},
		{
			name:     "Origin is nil",
			context:  BindingContext{Email: &email, Origin: nil},
			expected: "john.smith@email.com",
		},
		{
			name:     "Origin is empty",
			context:  BindingContext{Email: &email, Origin: &emptyStr},
			expected: "john.smith@email.com",
		},
		{
			name:     "Email is nil",
			context:  BindingContext{Email: nil, Origin: &origin},
			expected: "origin",
		},
		{
			name:     "Email is empty",
			context:  BindingContext{Email: &emptyStr, Origin: &origin},
			expected: "origin",
		},
		{
			name:     "Both Email and Origin are set",
			context:  BindingContext{Email: &email, Origin: &origin},
			expected: "john.smith@email.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.context.CreatedBy())
		})
	}
}
