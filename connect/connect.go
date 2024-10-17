package connect

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"pulsyflux/util"
	"sync"
	"time"
)

type Connection struct {
	server   *http.Server
	host     string
	port     int
	messages chan string
	mu       *sync.Mutex
}

var connections map[string]*Connection

func New(address string) (*Connection, error) {
	host, port, err := util.GetHostAndPortFromAddress(address)
	if err != nil {
		return nil, err
	}
	if connections == nil {
		connections = make(map[string]*Connection)
	}
	conn, exists := connections[address]
	if exists {
		return conn, nil
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		serverStoppedErr := fmt.Sprintf("failed to open a connection port %d: %s\n", port, err.Error())
		return nil, errors.New(serverStoppedErr)
	}
	conn = &Connection{}
	conn.host = host
	conn.port = port
	conn.mu = &sync.Mutex{}
	conn.messages = make(chan string)
	conn.server = &http.Server{
		Addr:           address,
		Handler:        conn,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	connections[address] = conn
	go conn.server.Serve(listener)
	fmt.Printf("server listening on port %d\n", port)
	return conn, nil
}

func (conn *Connection) Channel() chan string {
	return conn.messages
}

func (conn *Connection) ServeHTTP(response http.ResponseWriter, request *http.Request) {
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
	if len(base64Str) > 0 {
		conn.messages <- base64Str
	}
	response.WriteHeader(http.StatusOK)
}
