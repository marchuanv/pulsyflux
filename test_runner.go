package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("go", "test", "-run", "TestClientBidirectional", "-count=1", "-timeout=10s", "-v")
	cmd.Dir = "e:\\github\\pulsyflux\\socket"
	output, err := cmd.CombinedOutput()
	
	fmt.Println(string(output))
	
	if err != nil {
		fmt.Printf("Test failed: %v\n", err)
		os.Exit(1)
	}
	
	if strings.Contains(string(output), "PASS") {
		fmt.Println("\n✓ Test PASSED")
		os.Exit(0)
	} else {
		fmt.Println("\n✗ Test FAILED")
		os.Exit(1)
	}
}
