package connect

import (
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"
)

func testSendingMessageOnConnection(test *testing.T, address string) {
	expectedMsg := "Hello World"
	conn, err := OpenConnection(address)
	if err != nil {
		test.Fatal(err)
	}
	go (func() {
		response, err := Send(HTTPSchema, HttpGET, address, "/", expectedMsg)
		if err != nil {
			test.Error(err)
		}
		if response.StatusCode != http.StatusOK || response.StatusMessage != "200 OK" {
			test.Errorf("unexpected status code and message: %s", err)
		}
	})()
	for message := range conn.Messages {
		if message != expectedMsg {
			test.Fatalf("expected: \"%s\", but received \"%s\"", expectedMsg, message)
		}
	}
}

func TestMultipleConnections(test *testing.T) {
	runtime.GC()
	time.Sleep(5 * time.Second)
	var m1, m2 runtime.MemStats
	for counter := 1; counter < 11; counter++ {
		fmt.Printf("\r\nTest sending message on open connection and leaving it open:%d\r\n", counter)
		address := fmt.Sprintf("localhost:%d000", counter)
		testSendingMessageOnConnection(test, address)
		runtime.GC()
		runtime.ReadMemStats(&m1)
		fmt.Println("cumulative bytes allocated for heap objects:", m1.TotalAlloc)
		fmt.Println("cumulative count of heap objects allocated:", m1.Mallocs)
	}
	time.Sleep(5 * time.Second)
	runtime.GC()
	runtime.ReadMemStats(&m2)
	fmt.Println("\r\ncumulative bytes allocated for heap objects:", m2.TotalAlloc)
	fmt.Println("cumulative count of heap objects allocated:", m2.Mallocs)
}
