package main

// #include <stdlib.h>
import "C"
import (
	"sync"
	"unsafe"

	"github.com/google/uuid"
	"pulsyflux/broker"
)

type subscription struct {
	ch     <-chan []byte
	cancel chan struct{}
}

var (
	clients       = make(map[int]*broker.Client)
	servers       = make(map[int]*broker.Server)
	subscriptions = make(map[int]*subscription)
	nextID        = 1
	mu            sync.Mutex
)

//export ClientNew
func ClientNew(address *C.char, channelID *C.char) C.int {
	addr := C.GoString(address)
	chanID, err := uuid.Parse(C.GoString(channelID))
	if err != nil {
		return -1
	}

	client, err := broker.NewClient(addr, chanID)
	if err != nil {
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	clients[id] = client
	mu.Unlock()

	return C.int(id)
}

//export ClientPublish
func ClientPublish(id C.int, payload *C.char, payloadLen C.int) C.int {
	mu.Lock()
	client, ok := clients[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	payloadBytes := C.GoBytes(unsafe.Pointer(payload), payloadLen)

	if err := client.Publish(payloadBytes); err != nil {
		return -1
	}

	return 0
}

//export ClientSubscribe
func ClientSubscribe(id C.int) C.int {
	mu.Lock()
	client, ok := clients[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	ch := client.Subscribe()

	sub := &subscription{
		ch:     ch,
		cancel: make(chan struct{}),
	}

	mu.Lock()
	subID := nextID
	nextID++
	subscriptions[subID] = sub
	mu.Unlock()

	return C.int(subID)
}

//export SubscriptionReceive
func SubscriptionReceive(subID C.int, payload **C.char, payloadLen *C.int) C.int {
	mu.Lock()
	sub, ok := subscriptions[int(subID)]
	mu.Unlock()

	if !ok {
		return -1
	}

	select {
	case msg, ok := <-sub.ch:
		if !ok {
			return -2 // Channel closed
		}

		*payload = (*C.char)(C.CBytes(msg))
		*payloadLen = C.int(len(msg))

		return 0
	case <-sub.cancel:
		return -2
	default:
		return -3 // No message available
	}
}

//export SubscriptionClose
func SubscriptionClose(subID C.int) C.int {
	mu.Lock()
	sub, ok := subscriptions[int(subID)]
	if ok {
		delete(subscriptions, int(subID))
	}
	mu.Unlock()

	if !ok {
		return -1
	}

	close(sub.cancel)
	return 0
}

//export ServerNew
func ServerNew(address *C.char) C.int {
	s := broker.NewServer(C.GoString(address))

	mu.Lock()
	id := nextID
	nextID++
	servers[id] = s
	mu.Unlock()

	return C.int(id)
}

//export ServerStart
func ServerStart(id C.int) C.int {
	mu.Lock()
	s, ok := servers[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	if err := s.Start(); err != nil {
		return -1
	}
	return 0
}

//export ServerAddr
func ServerAddr(id C.int) *C.char {
	mu.Lock()
	s, ok := servers[int(id)]
	mu.Unlock()

	if !ok {
		return nil
	}

	return C.CString(s.Addr())
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

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

func main() {}
