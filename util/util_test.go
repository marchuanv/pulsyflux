package util

import (
	"testing"
)

func TestIsValidUUID(test *testing.T) {
	isValid := IsValidUUID("831d2c15-1a90-413e-9189-4cff18b5ae9c")
	if !isValid {
		test.Fail()
	}
}
func TestIsNotValidUUID(test *testing.T) {
	isValid := IsValidUUID("831d2c15-1a90-aaaa-9189-4cff18b5ae9c")
	if isValid {
		test.Fail()
	}
}
