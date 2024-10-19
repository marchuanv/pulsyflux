package connect

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"pulsyflux/util"
	"time"
)

type state int

func (st state) String() string {
	switch st {
	case CLOSED:
		return "CLOSED"
	case OPEN:
		return "OPEN"
	case CREATED:
		return "CREATED"
	case ERROR:
		return "ERROR"
	default:
		return fmt.Sprintf("%d", int(st))
	}
}

const (
	OPEN state = iota
	CLOSED
	CREATED
	ERROR
)

type Connection struct {
	server   *http.Server
	host     string
	port     int
	state    state
	address  string
	messages chan string
}

var connections = make(map[string]*Connection)

func New(address string) (*Connection, error) {
	host, port, err := util.GetHostAndPortFromAddress(address)
	if err != nil {
		return nil, err
	}
	if len(connections) >= 10 {
		cleanup()
		if len(connections) >= 10 {
			return nil, errors.New("connection limit was reached")
		}
	}
	conn, exists := connections[address]
	if exists {
		return conn, nil
	}
	conn = &Connection{}
	conn.host = host
	conn.port = port
	conn.state = CREATED
	conn.address = address
	connections[address] = conn
	go (func() {
		time.Sleep(1 * time.Second)
		cleanup()
	})()
	return conn, nil
}

func (conn *Connection) GetMessage() (string, error) {
	if conn.state != OPEN {
		return "", errors.New("Connection channel is closed")
	}
	return <-conn.messages, nil
}

func (conn *Connection) Open() error {
	if conn.state == OPEN || conn.state == ERROR {
		return nil
	}
	go func() {
		err := newServer(conn)
		if err != nil {
			fmt.Printf("failed to create a new http server: %v", err)
			conn.state = ERROR
		}
		listener, err := net.Listen("tcp", conn.address)
		if err != nil {
			fmt.Printf("failed to listen on port %d: %v", conn.port, err)
			conn.state = ERROR
			listener.Close()
		} else {
			err = conn.server.Serve(listener)
			listener.Close()
			if err == http.ErrServerClosed {
				conn.state = CLOSED
			} else if err != nil {
				conn.state = ERROR
				fmt.Printf("failed to listen on port %d: %v", conn.port, err)
			}
		}
		conn.server = nil
		close(conn.messages)
		conn.messages = nil
		listener.Close()
	}()
	res, err := Send(HTTPSchema, "GET", conn.address, "/", "")
	if err != nil {
		return err
	}
	if res.statusCode != http.StatusServiceUnavailable {
		return errors.New("expected service is not available right now")
	}
	conn.state = OPEN
	conn.messages = make(chan string)
	fmt.Printf("server is listening on port %d\n", conn.port)
	return nil
}

func (conn *Connection) Close() {
	if conn.state != OPEN {
		return
	}
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()
	err := conn.server.Shutdown(shutdownCtx)
	if err != nil {
		fmt.Printf("HTTP shutdown error: %v", err)
	} else {
		fmt.Println("Graceful shutdown complete.")
	}
}

func newServer(conn *Connection) error {
	_, _, err := util.GetHostAndPortFromAddress(conn.address)
	if err != nil {
		return errors.New("connection does not have the correct address")
	}
	conn.server = &http.Server{
		Addr:           conn.address,
		Handler:        conn,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return nil
}

func (conn *Connection) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if conn.state != OPEN {
		response.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	resBody, err := util.StringFromReader(request.Body)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	conn.messages <- resBody
	response.WriteHeader(http.StatusOK)
}

func cleanup() {
	for connId := range maps.Keys(connections) {
		conn := connections[connId]
		if conn.state == ERROR || conn.state == CLOSED {
			fmt.Printf("\r\nRemoved connections with states: \"%s\"|\"%s\", close connections that are not used.\r\n", state(ERROR), state(CLOSED))
			delete(connections, connId)
		}
	}
}
