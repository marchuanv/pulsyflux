package connect

import (
	"testing"
)

func TestServer(test *testing.T) {
	_, err := Subscribe()
	if err != nil {
		test.Fail()
	}
}
