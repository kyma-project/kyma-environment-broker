package rules

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRule_SignatureWithValues(t *testing.T) {
	// given
	r := NewRule()
	r.Plan = "gcp"
	r.PlatformRegion = "cf-sa30"

	// when
	signature := r.SignatureWithValues()

	// then
	assert.Equal(t, "gcp(PR=cf-sa30,HR=)", signature)
}

func TestRule_MirroredSignature(t *testing.T) {
	// given
	r := NewRule()
	r.Plan = "gcp"
	r.PlatformRegion = "cf-sa30"

	// when
	signature := r.MirroredSignature()

	// then
	assert.Equal(t, "gcp(PR=,HR=attr)", signature)
}
