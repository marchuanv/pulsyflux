package msgbus

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"pulsyflux/message"
	"strconv"
	"time"
)

type connection struct {
	server *http.Server
	queue  *MsgQueue
	host   string
	port   int
}

type MsgQueue struct {
	messages []*message.Message
}

func MessageQueue(msgBus *MsgBus) (*MsgQueue, error) {
	portStr := strconv.Itoa(msgBus.port)
	address := net.JoinHostPort(msgBus.host, portStr)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		serverStoppedErr := fmt.Sprintf("failed to create message queue on port %d: %s\n", msgBus.port, err.Error())
		return nil, errors.New(serverStoppedErr)
	}
	conn := &connection{}
	conn.host = msgBus.host
	conn.port = msgBus.port
	conn.queue = &MsgQueue{}
	conn.server = &http.Server{
		Addr:           address,
		Handler:        conn,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go conn.server.Serve(listener)
	fmt.Printf("message queue created on port %d\n", msgBus.port)
	return conn.queue, nil
}

func (conn *connection) ServeHTTP(response http.ResponseWriter, request *http.Request) {
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
	msg, err := message.NewDeserialiseMessage(base64Str)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	channel := msg.Channel()
	text := msg.Text()
	msgFromAddress := msg.FromAddress()
	if msgFromAddress.Host == conn.host && msgFromAddress.Port == conn.port { //from this server
		if channel == "server" && text == "stop" {
			msg.Dispose()
			err = conn.server.Close()
			if err == nil {
				fmt.Print("stopping server")
			} else {
				fmt.Printf("failed to stop message queue: %s.\n", err.Error())
				response.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else if channel == "queue" && text == "send" {
			if len(conn.queue.messages) > 0 {

				msgToSend := conn.queue.messages[0]
				msgToSendToAddress := msgToSend.ToAddress()
				msgToSendToChannel := msgToSend.Channel()

				if msgToSendToAddress.Host != conn.host && msgToSendToAddress.Port != conn.port {

					messages := conn.queue.messages[1:]
					conn.queue.messages = nil
					conn.queue.messages = messages

					serialised, err := msgToSend.Serialise()
					if err != nil {
						response.WriteHeader(http.StatusInternalServerError)
						return
					}
					body := []byte(serialised)
					url := fmt.Sprintf("http://%s:%d/%s/message", msgToSendToAddress.Host, msgToSendToAddress.Port, msgToSendToChannel)
					res, err := http.Post(url, "text/plain", bytes.NewBuffer(body))
					if err != nil {
						response.WriteHeader(http.StatusInternalServerError)
						return
					}
					if res.StatusCode < 200 || res.StatusCode > 299 {
						fmt.Printf("Response StatusCode:%d, StatusMessage:%s", res.StatusCode, res.Status)
						response.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			}
		}
	} else {
		conn.queue.messages = append(conn.queue.messages, msg)
		fmt.Printf("message is queued.\n")
	}
}
