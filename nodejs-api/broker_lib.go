package main

// #include <stdlib.h>
import "C"
import (
	"unsafe"

	"github.com/google/uuid"
	"pulsyflux/broker"
)

var (
	servers = make(map[int]*broker.Server)
	clients = make(map[int]*broker.Client)
	subs    = make(map[int]<-chan []byte)
	nextID  = 1
)

//export ServerNew
func ServerNew(address *C.char) C.int {
	s := broker.NewServer(C.GoString(address))
	id := nextID
	nextID++
	servers[id] = s
	return C.int(id)
}

//export ServerStart
func ServerStart(id C.int) C.int {
	s := servers[int(id)]
	if s == nil {
		return -1
	}
	if err := s.Start(); err != nil {
		return -1
	}
	return 0
}

//export ServerAddr
func ServerAddr(id C.int) *C.char {
	s := servers[int(id)]
	if s == nil {
		return nil
	}
	return C.CString(s.Addr())
}

//export ServerStop
func ServerStop(id C.int) C.int {
	s := servers[int(id)]
	if s == nil {
		return -1
	}
	if err := s.Stop(); err != nil {
		return -1
	}
	delete(servers, int(id))
	return 0
}

//export NewClient
func NewClient(address *C.char, channelID *C.char) C.int {
	chanID, err := uuid.Parse(C.GoString(channelID))
	if err != nil {
		return -1
	}
	client, err := broker.NewClient(C.GoString(address), chanID)
	if err != nil {
		return -1
	}
	id := nextID
	nextID++
	clients[id] = client
	return C.int(id)
}

//export Publish
func Publish(id C.int, payload *C.char, payloadLen C.int) C.int {
	client := clients[int(id)]
	if client == nil {
		return -1
	}
	data := C.GoBytes(unsafe.Pointer(payload), payloadLen)
	if err := client.Publish(data); err != nil {
		return -1
	}
	return 0
}

//export Subscribe
func Subscribe(id C.int, payload **C.char, payloadLen *C.int) C.int {
	client := clients[int(id)]
	if client == nil {
		return -1
	}
	
	// Get or create subscription channel
	ch, exists := subs[int(id)]
	if !exists {
		ch = client.Subscribe()
		subs[int(id)] = ch
	}
	
	// Receive from channel
	select {
	case msg, ok := <-ch:
		if !ok {
			return -2
		}
		*payload = (*C.char)(C.CBytes(msg))
		*payloadLen = C.int(len(msg))
		return 0
	default:
		return -3
	}
}

//export FreePayload
func FreePayload(ptr unsafe.Pointer) {
	C.free(ptr)
}

//export Cleanup
func Cleanup() {
	// Clean up all resources
	for id := range servers {
		if s := servers[id]; s != nil {
			s.Stop()
		}
	}
	for id := range clients {
		delete(clients, id)
	}
	for id := range subs {
		delete(subs, id)
	}
	servers = make(map[int]*broker.Server)
	clients = make(map[int]*broker.Client)
	subs = make(map[int]<-chan []byte)
}

func main() {}
