package main

// #include <stdlib.h>
import "C"
import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"
	"unsafe"

	pfSocket "pulsyflux/socket"

	"github.com/google/uuid"
)

type (
	consumer = pfSocket.Consumer
	provider = pfSocket.Provider
	server   = pfSocket.Server
)

var (
	consumers = make(map[int]*consumer)
	providers = make(map[int]*provider)
	servers   = make(map[int]*server)
	nextID    = 1
	mu        sync.Mutex
)

//export ConsumerNew
func ConsumerNew(address *C.char, channelID *C.char) C.int {
	addr := C.GoString(address)
	chanID, err := uuid.Parse(C.GoString(channelID))
	if err != nil {
		return -1
	}

	c, err := pfSocket.NewConsumer(addr, chanID)
	if err != nil {
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	consumers[id] = c
	mu.Unlock()

	return C.int(id)
}

//export ConsumerSend
func ConsumerSend(id C.int, data *C.char, dataLen C.int, timeoutMs C.int) *C.char {
	mu.Lock()
	c, ok := consumers[int(id)]
	mu.Unlock()

	if !ok {
		return C.CString("consumer not found")
	}

	payload := C.GoBytes(unsafe.Pointer(data), dataLen)
	timeout := time.Duration(timeoutMs) * time.Millisecond

	resp, err := c.Send(bytes.NewReader(payload), timeout)
	if err != nil {
		return C.CString(err.Error())
	}

	respData, err := io.ReadAll(resp)
	if err != nil {
		return C.CString(err.Error())
	}

	return C.CString(string(respData))
}

//export ConsumerClose
func ConsumerClose(id C.int) C.int {
	mu.Lock()
	c, ok := consumers[int(id)]
	if ok {
		delete(consumers, int(id))
	}
	mu.Unlock()

	if !ok {
		return -1
	}

	if err := c.Close(); err != nil {
		return -1
	}
	return 0
}

//export ProviderNew
func ProviderNew(address *C.char, channelID *C.char) C.int {
	addr := C.GoString(address)
	chanID, err := uuid.Parse(C.GoString(channelID))
	if err != nil {
		return -1
	}

	p, err := pfSocket.NewProvider(addr, chanID)
	if err != nil {
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	providers[id] = p
	mu.Unlock()

	return C.int(id)
}

//export ProviderReceive
func ProviderReceive(id C.int, reqID **C.char, data **C.char, dataLen *C.int) C.int {
	mu.Lock()
	p, ok := providers[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	// Use a very short timeout to make it non-blocking
	done := make(chan struct{})
	var uuid uuid.UUID
	var reader io.Reader
	var success bool

	go func() {
		uuid, reader, success = p.Receive()
		close(done)
	}()

	select {
	case <-done:
		if !success {
			return -1
		}
	case <-time.After(1 * time.Millisecond):
		return -2 // No request available (timeout)
	}

	payload, err := io.ReadAll(reader)
	if err != nil {
		return -1
	}

	*reqID = C.CString(uuid.String())
	*data = C.CString(string(payload))
	*dataLen = C.int(len(payload))

	return 0
}

//export ProviderRespond
func ProviderRespond(id C.int, reqID *C.char, data *C.char, dataLen C.int, errMsg *C.char) C.int {
	mu.Lock()
	p, ok := providers[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	uuid, err := uuid.Parse(C.GoString(reqID))
	if err != nil {
		return -1
	}

	var respErr error
	if errMsg != nil {
		respErr = errors.New(C.GoString(errMsg))
	}

	var reader io.Reader
	if data != nil && dataLen > 0 {
		payload := C.GoBytes(unsafe.Pointer(data), dataLen)
		reader = bytes.NewReader(payload)
	}

	p.Respond(uuid, reader, respErr)
	return 0
}

//export ProviderClose
func ProviderClose(id C.int) C.int {
	mu.Lock()
	p, ok := providers[int(id)]
	if ok {
		delete(providers, int(id))
	}
	mu.Unlock()

	if !ok {
		return -1
	}

	if err := p.Close(); err != nil {
		return -1
	}
	return 0
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

//export ServerNew
func ServerNew(port *C.char) C.int {
	s := pfSocket.NewServer(C.GoString(port))
	if err := s.Start(); err != nil {
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	servers[id] = s
	mu.Unlock()

	return C.int(id)
}

//export ServerStop
func ServerStop(id C.int) C.int {
	mu.Lock()
	s, ok := servers[int(id)]
	if ok {
		delete(servers, int(id))
	}
	mu.Unlock()

	if !ok {
		return -1
	}

	if err := s.Stop(); err != nil {
		return -1
	}
	return 0
}

func main() {}
