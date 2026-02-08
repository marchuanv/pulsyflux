package main

// #include <stdlib.h>
import "C"
import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"pulsyflux/messagebus"
)

type (
	bus    = messagebus.Bus
	server = messagebus.Server
)

type subscription struct {
	ch     <-chan *messagebus.Message
	cancel chan struct{}
}

var (
	buses         = make(map[int]bus)
	servers       = make(map[int]*server)
	subscriptions = make(map[int]*subscription)
	nextID        = 1
	mu            sync.Mutex
)

//export BusNew
func BusNew(address *C.char, channelID *C.char) C.int {
	addr := C.GoString(address)
	chanID, err := uuid.Parse(C.GoString(channelID))
	if err != nil {
		return -1
	}

	b, err := messagebus.NewBus(addr, chanID)
	if err != nil {
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	buses[id] = b
	mu.Unlock()

	return C.int(id)
}

//export BusPublish
func BusPublish(id C.int, topic *C.char, payload *C.char, payloadLen C.int, headers *C.char, timeoutMs C.int) C.int {
	mu.Lock()
	b, ok := buses[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	topicStr := C.GoString(topic)
	payloadBytes := C.GoBytes(unsafe.Pointer(payload), payloadLen)

	var headerMap map[string]string
	if headers != nil {
		headersStr := C.GoString(headers)
		if err := json.Unmarshal([]byte(headersStr), &headerMap); err != nil {
			return -1
		}
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := b.Publish(ctx, topicStr, payloadBytes, headerMap); err != nil {
		return -1
	}

	return 0
}

//export BusSubscribe
func BusSubscribe(id C.int, topic *C.char) C.int {
	mu.Lock()
	b, ok := buses[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	topicStr := C.GoString(topic)
	ch, err := b.Subscribe(topicStr)
	if err != nil {
		return -1
	}

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

//export BusReceive
func BusReceive(subID C.int, msgID **C.char, topic **C.char, payload **C.char, payloadLen *C.int, headers **C.char, timestamp *C.longlong) C.int {
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

		*msgID = C.CString(msg.ID.String())
		*topic = C.CString(msg.Topic)
		*payload = (*C.char)(C.CBytes(msg.Payload))
		*payloadLen = C.int(len(msg.Payload))
		*timestamp = C.longlong(msg.Timestamp.Unix())

		if len(msg.Headers) > 0 {
			headersJSON, _ := json.Marshal(msg.Headers)
			*headers = C.CString(string(headersJSON))
		} else {
			*headers = nil
		}

		return 0
	case <-sub.cancel:
		return -2
	default:
		return -3 // No message available
	}
}

//export BusUnsubscribe
func BusUnsubscribe(id C.int, topic *C.char) C.int {
	mu.Lock()
	b, ok := buses[int(id)]
	mu.Unlock()

	if !ok {
		return -1
	}

	topicStr := C.GoString(topic)
	if err := b.Unsubscribe(topicStr); err != nil {
		return -1
	}

	return 0
}

//export BusClose
func BusClose(id C.int) C.int {
	mu.Lock()
	b, ok := buses[int(id)]
	if ok {
		delete(buses, int(id))
	}
	mu.Unlock()

	if !ok {
		return -1
	}

	if err := b.Close(); err != nil {
		return -1
	}
	return 0
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
func ServerNew(port *C.char) C.int {
	s := messagebus.NewServer(C.GoString(port))

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
