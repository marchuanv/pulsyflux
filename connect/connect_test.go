package connect

import (
	"fmt"
	"net/http"
	"pulsyflux/util"
	"runtime"
	"testing"
	"time"
)

func testSendingMessageOnClosedConnection(test *testing.T, address string) {
	expectedMsg := ""
	conn, err := New(address)
	if err != nil {
		test.Fatal(err)
	}
	err = conn.Open()
	if err != nil {
		test.Fatal(err)
	}
	conn.Close()
	response, err := Send(HTTPSchema, http.MethodGet, address, "/", "")
	if err == nil {
		test.Fatalf("Get \"%s\": No connection could be made because the target machine actively refused it.", address)
	}
	if response.statusCode > 0 || util.IsEmptyString(response.statusMessage) {
		test.Fatalf("unexpected status code and message: %s", err)
	}
	msg, err := conn.GetMessage()
	if err == nil {
		test.Fatal("should not receive messages from a closed channel")
	}
	if msg != expectedMsg {
		test.Fatal("should not receive messages from a closed channel")
	}
}

func testSendingMessageOnOpenConnection(test *testing.T, address string) {
	expectedMsg := "Hello World"
	conn, err := New(address)
	if err != nil {
		test.Fatal(err)
	}
	err = conn.Open()
	if err != nil {
		test.Fatal(err)
	}
	go (func() {
		response, err := Send(HTTPSchema, http.MethodGet, address, "/", expectedMsg)
		if err != nil {
			test.Error(err)
		}
		if response.statusCode != http.StatusOK || response.statusMessage != "200 OK" {
			test.Errorf("unexpected status code and message: %s", err)
		}
	})()
	msg, err := conn.GetMessage()
	if err != nil {
		test.Fatal(err)
	}
	if msg != expectedMsg {
		test.Fatalf("expected: \"%s\", but received \"%s\"", expectedMsg, msg)
	}
}

func TestMultipleOpenConnections(test *testing.T) {
	var m1, m2 runtime.MemStats
	for counter := 1; counter < 11; counter++ {
		fmt.Printf("\r\nTest sending message on open connection and leaving it open:%d\r\n", counter)
		address := fmt.Sprintf("localhost:%d000", counter)
		testSendingMessageOnOpenConnection(test, address)
		runtime.GC()
		runtime.ReadMemStats(&m1)
		fmt.Println("cumulative bytes allocated for heap objects:", m1.TotalAlloc)
		fmt.Println("cumulative count of heap objects allocated:", m1.Mallocs)
	}
	time.Sleep(10 * time.Second)
	runtime.GC()
	runtime.ReadMemStats(&m2)
	fmt.Println("cumulative bytes allocated for heap objects:", m2.TotalAlloc)
	fmt.Println("cumulative count of heap objects allocated:", m2.Mallocs)
}

func TestMultipleClosedConnections(test *testing.T) {
	var m1, m2 runtime.MemStats
	for counter := 1; counter < 21; counter++ {
		fmt.Printf("\r\nTest sending message on open connection and closing it afterwards:%d\r\n", counter)
		address := fmt.Sprintf("localhost:%d000", counter)
		testSendingMessageOnClosedConnection(test, address)
		runtime.GC()
		runtime.ReadMemStats(&m1)
		fmt.Println("cumulative bytes allocated for heap objects:", m1.TotalAlloc)
		fmt.Println("cumulative count of heap objects allocated:", m1.Mallocs)
	}
	time.Sleep(10 * time.Second)
	runtime.GC()
	runtime.ReadMemStats(&m2)
	fmt.Println("cumulative bytes allocated for heap objects:", m2.TotalAlloc)
	fmt.Println("cumulative count of heap objects allocated:", m2.Mallocs)
}
