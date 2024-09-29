package uuid

import (
	"errors"
	"testing"
)

func testUUIDCtor(t *testing.T) {
	err := errors.New("Failing Test")
	t.Fatalf(`Hello("") = %q, %v, want "", error`, "awdawdawdwa", err)
}
