package main

// #include <stdlib.h>
import "C"
import (
	"bytes"
	"errors"
	"io"
	"log"
	"sync"
	"time"
	"unsafe"

	"pulsyflux/socket"

	"github.com/google/uuid"
)

type (
	consumer = socket.Consumer
	provider = socket.Provider
	server   = socket.Server
)

var (
	consumers = make(map[int]*consumer)
	providers = make(map[int]*provider)
	servers   = make(map[int]*server)
	nextID    = 1
	mu        sync.RWMutex
)

//export ConsumerNew
func ConsumerNew(address *C.char, channelID *C.char) C.int {
	addr := C.GoString(address)
	chanIDStr := C.GoString(channelID)
	log.Printf("Creating consumer: address=%s, channelID=%s", addr, chanIDStr)
	
	chanID, err := uuid.Parse(chanIDStr)
	if err != nil {
		log.Printf("Failed to parse channelID %s: %v", chanIDStr, err)
		return -1
	}

	c, err := socket.NewConsumer(addr, chanID)
	if err != nil {
		log.Printf("Failed to create consumer for %s: %v", addr, err)
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	consumers[id] = c
	mu.Unlock()

	log.Printf("Consumer created successfully: id=%d, address=%s", id, addr)
	return C.int(id)
}

//export ConsumerSend
func ConsumerSend(id C.int, data *C.char, dataLen C.int, timeoutMs C.int) *C.char {
	log.Printf("Consumer send: id=%d, dataLen=%d, timeout=%dms", id, dataLen, timeoutMs)
	
	mu.RLock()
	c, ok := consumers[int(id)]
	mu.RUnlock()

	if !ok {
		log.Printf("Consumer %d not found", id)
		return C.CString("consumer not found")
	}

	payload := C.GoBytes(unsafe.Pointer(data), dataLen)
	timeout := time.Duration(timeoutMs) * time.Millisecond

	log.Printf("Sending request: consumer=%d, payloadSize=%d", id, len(payload))
	resp, err := c.Send(bytes.NewReader(payload), timeout)
	if err != nil {
		log.Printf("Consumer %d send failed: %v", id, err)
		return C.CString(err.Error())
	}

	respData, err := io.ReadAll(resp)
	if err != nil {
		log.Printf("Consumer %d response read failed: %v", id, err)
		return C.CString(err.Error())
	}

	log.Printf("Consumer %d send successful: responseSize=%d", id, len(respData))
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

	log.Printf("Closing consumer %d", id)
	if err := c.Close(); err != nil {
		log.Printf("Error closing consumer %d: %v", id, err)
		return -1
	}
	log.Printf("Consumer %d closed successfully", id)
	return 0
}

//export ProviderNew
func ProviderNew(address *C.char, channelID *C.char) C.int {
	addr := C.GoString(address)
	chanIDStr := C.GoString(channelID)
	log.Printf("Creating provider: address=%s, channelID=%s", addr, chanIDStr)
	
	chanID, err := uuid.Parse(chanIDStr)
	if err != nil {
		log.Printf("Failed to parse channelID %s: %v", chanIDStr, err)
		return -1
	}

	p, err := socket.NewProvider(addr, chanID)
	if err != nil {
		log.Printf("Failed to create provider for %s: %v", addr, err)
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	providers[id] = p
	mu.Unlock()

	log.Printf("Provider created successfully: id=%d, address=%s", id, addr)
	return C.int(id)
}

//export ProviderReceive
func ProviderReceive(id C.int, reqID **C.char, data **C.char, dataLen *C.int) C.int {
	mu.RLock()
	p, ok := providers[int(id)]
	mu.RUnlock()

	if !ok {
		log.Printf("Provider %d not found", id)
		return -1
	}

	// Direct non-blocking call
	uuid, reader, success := p.Receive()
	if !success {
		// Don't log this as it's expected when no requests are available
		return -2 // No request available
	}

	payload, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("Provider %d failed to read request payload: %v", id, err)
		return -1
	}

	*reqID = C.CString(uuid.String())
	*data = (*C.char)(C.CBytes(payload))
	*dataLen = C.int(len(payload))

	log.Printf("Provider %d received request: reqID=%s, payloadSize=%d", id, uuid.String(), len(payload))
	return 0
}

//export ProviderRespond
func ProviderRespond(id C.int, reqID *C.char, data *C.char, dataLen C.int, errMsg *C.char) C.int {
	reqIDStr := C.GoString(reqID)
	log.Printf("Provider respond: id=%d, reqID=%s, dataLen=%d", id, reqIDStr, dataLen)
	
	mu.RLock()
	p, ok := providers[int(id)]
	mu.RUnlock()

	if !ok {
		log.Printf("Provider %d not found", id)
		return -1
	}

	uuid, err := uuid.Parse(reqIDStr)
	if err != nil {
		log.Printf("Failed to parse reqID %s: %v", reqIDStr, err)
		return -1
	}

	var respErr error
	if errMsg != nil {
		errorStr := C.GoString(errMsg)
		respErr = errors.New(errorStr)
		log.Printf("Provider %d responding with error: %s", id, errorStr)
	}

	var reader io.Reader
	if data != nil && dataLen > 0 {
		payload := C.GoBytes(unsafe.Pointer(data), dataLen)
		reader = bytes.NewReader(payload)
		log.Printf("Provider %d responding with data: payloadSize=%d", id, len(payload))
	}

	p.Respond(uuid, reader, respErr)
	log.Printf("Provider %d response sent successfully for reqID=%s", id, reqIDStr)
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

	log.Printf("Closing provider %d", id)
	if err := p.Close(); err != nil {
		log.Printf("Error closing provider %d: %v", id, err)
		return -1
	}
	log.Printf("Provider %d closed successfully", id)
	return 0
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

//export ServerNew
func ServerNew(port *C.char) C.int {
	portStr := C.GoString(port)
	log.Printf("Creating server on port: %s", portStr)
	
	s := socket.NewServer(portStr)

	mu.Lock()
	id := nextID
	nextID++
	servers[id] = s
	mu.Unlock()

	log.Printf("Server created: id=%d, port=%s", id, portStr)
	return C.int(id)
}

//export ServerStart
func ServerStart(id C.int) C.int {
	mu.RLock()
	s, ok := servers[int(id)]
	mu.RUnlock()

	if !ok {
		return -1
	}

	log.Printf("Starting server %d", id)
	if err := s.Start(); err != nil {
		log.Printf("Failed to start server %d: %v", id, err)
		return -1
	}
	log.Printf("Server %d started successfully", id)
	return 0
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

	log.Printf("Stopping server %d", id)
	if err := s.Stop(); err != nil {
		log.Printf("Error stopping server %d: %v", id, err)
		return -1
	}
	log.Printf("Server %d stopped successfully", id)
	return 0
}

func main() {}