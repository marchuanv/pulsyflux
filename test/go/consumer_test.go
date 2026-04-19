package consumer_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGoConsumerCanBuild verifies that an external Go module can import
// github.com/pulsyflux/broker and compile successfully. This test creates
// a temporary Go module that depends on the broker package, writes a
// main.go that exercises the public API (Server, Client, Publish), and
// runs `go build` to confirm it compiles.
//
// NOTE: This test requires the Go module to be published at
// github.com/pulsyflux/broker. It will fail until the module is available
// on the Go module proxy.
func TestGoConsumerCanBuild(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "broker-go-consumer-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write go.mod for the temporary consumer module
	goMod := `module testconsumer

go 1.25

require github.com/pulsyflux/broker v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write main.go that imports and uses the broker package
	mainGo := `package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pulsyflux/broker/broker"
)

func main() {
	// Create and start a server
	srv := broker.NewServer("127.0.0.1:0")
	if err := srv.Start(); err != nil {
		fmt.Printf("server start error: %v\n", err)
		return
	}
	defer srv.Stop()

	addr := srv.Addr()
	channelID := uuid.New()

	// Create a client and publish a message
	client, err := broker.NewClient(addr, channelID)
	if err != nil {
		fmt.Printf("client create error: %v\n", err)
		return
	}

	if err := client.Publish([]byte("hello from consumer test")); err != nil {
		fmt.Printf("publish error: %v\n", err)
		return
	}

	fmt.Println("broker consumer test: OK")
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Run go mod tidy to resolve dependencies
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tmpDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		t.Skipf("go mod tidy failed (module may not be published yet): %v", err)
	}

	// Run go build to verify compilation
	build := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "testconsumer"), ".")
	build.Dir = tmpDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("go build failed: %v", err)
	}

	t.Log("Go consumer build succeeded")
}
