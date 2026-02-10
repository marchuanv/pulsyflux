package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	for i := 1; i <= 10; i++ {
		fmt.Printf("\n=== Run %d/10 ===\n", i)
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, "go", "test", "-v", "-run", "TestClientBidirectional", "./socket")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		err := cmd.Run()
		cancel()
		
		if err != nil {
			fmt.Printf("\n❌ Test failed on run %d: %v\n", i, err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Run %d passed\n", i)
		time.Sleep(100 * time.Millisecond)
	}
	
	fmt.Println("\n🎉 All 10 runs passed!")
}
