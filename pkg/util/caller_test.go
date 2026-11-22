package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type caller struct{}

func (caller) method() string { return GetPackageAndFunctionName() }

func TestGetPackageAndFunctionName(t *testing.T) {
	name := GetPackageAndFunctionName()
	assert.Contains(t, name, "TestGetPackageAndFunctionName")
	assert.Contains(t, name, "util.")

	assert.Equal(t, "util.caller.method", caller{}.method())
}
