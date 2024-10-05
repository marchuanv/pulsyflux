package msgbus

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"net/http"
	"pulsyflux/message"
	"strconv"
	"sync"
	"time"
)

type MessageBus struct {
	mu     *sync.Mutex
	queue  []*message.Message
	server *http.Server
	host   string
	port   int
}

func New(host string, port int) (*MessageBus, error) {
	msgBus := MessageBus{
		&sync.Mutex{},
		[]*message.Message{},
		nil,
		host,
		port,
	}
	portStr := strconv.Itoa(msgBus.port)
	address := net.JoinHostPort(msgBus.host, portStr)
	msgBus.server = &http.Server{
		Addr:           address,
		Handler:        &msgBus,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return &msgBus, nil
}

// non blocking thread to start the server
func (msgBus *MessageBus) Start() error {
	msgBus.mu.Lock()
	listener, err := net.Listen("tcp", msgBus.server.Addr)
	if err != nil {
		msgBus.mu.Unlock()
		msgBus.server.Close()
		serverStoppedErr := fmt.Sprintf("messagebus not started on port %d: %s\n", msgBus.port, err.Error())
		return errors.New(serverStoppedErr)
	}
	go msgBus.server.Serve(listener)
	fmt.Printf("messagebus started on port %d\n", msgBus.port)
	msgBus.mu.Unlock()
	return nil
}
func (msgBus *MessageBus) Stop() error {
	msgBus.mu.Lock()
	defer msgBus.mu.Unlock()
	err := msgBus.server.Close()
	if err != nil {
		serverStoppedErr := fmt.Sprintf("failed to stop messagebus: %s.\n", err.Error())
		return errors.New(serverStoppedErr)
	}
	return nil
}
func (msgBus *MessageBus) Dequeue() *message.Message {
	if len(msgBus.queue) > 0 {
		msgBus.mu.Lock()
		defer msgBus.mu.Unlock()
		msg := msgBus.queue[0]
		queue := msgBus.queue[1:]
		msgBus.queue = nil
		msgBus.queue = queue
		return msg
	}
	return nil
}
func (msgBus *MessageBus) Enqueue(message *message.Message) {
	msgBus.mu.Lock()
	defer msgBus.mu.Unlock()
	msgBus.queue = append(msgBus.queue, message)
	fmt.Printf("message is queued.\n")
}
func (msgBus *MessageBus) Send(url string, message *message.Message) error {
	serialisedMsg, err := message.Serialise()
	if err != nil {
		return err
	}
	body := []byte(serialisedMsg)
	res, err := http.Post(url, "text/plain", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		errMsg := fmt.Sprintf("Response StatusCode:%d, StatusMessage:%s", res.StatusCode, res.Status)
		return errors.New(errMsg)
	}
	return nil
}
func (msgBus *MessageBus) ServeHTTP(response http.ResponseWriter, request *http.Request) {
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
