package msgbus

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net/http"
	"pulsyflux/message"
	"sync"
	"time"
)

type MessageBus struct {
	mu     sync.Mutex
	queue  []*message.Message
	server *http.Server
}

func New(port int) *MessageBus {
	msgBus := MessageBus{
		sync.Mutex{},
		[]*message.Message{},
		&http.Server{
			Addr:           fmt.Sprintf("localhost:%d", port),
			Handler:        new(MessageBus),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}
	return &msgBus
}
func (msgBus *MessageBus) Start() {
	execute := sync.OnceFunc(func() {
		fmt.Print("messagebus started\n")
		err := msgBus.server.ListenAndServe()
		if err != nil {
			fmt.Print(err.Error())
		}
	})
	go execute()
}
func (msgBus *MessageBus) Dequeue() *message.Message {
	if len(msgBus.queue) > 0 {
		msg := msgBus.queue[0] // The first element is the one to be dequeued.
		msgBus.queue[0] = nil
		msgBus.queue = msgBus.queue[1:]
		return msg
	}
	return nil
}
func (msgBus *MessageBus) Enqueue(message *message.Message) {
	msgBus.queue = append(msgBus.queue, message)
	fmt.Printf("message was queued.")
}
func (msgBus *MessageBus) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	msgBus.mu.Lock()
	defer msgBus.mu.Unlock()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(request.Body)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = request.Body.Close()
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	base64Str := buf.String()
	msg := message.Message{}
	base64Bytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	base64BytesBuf := bytes.Buffer{}
	base64BytesBuf.Write(base64Bytes)
	decoder := gob.NewDecoder(&base64BytesBuf)
	err = decoder.Decode(&msg)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/plain")
	response.WriteHeader(http.StatusOK)
	_, err = response.Write([]byte("messages queued"))
	if err != nil {
		fmt.Print(err.Error())
	}
	msgBus.Enqueue(&msg)
}
