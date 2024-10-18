package connect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"pulsyflux/util"
	"syscall"
	"time"
)

type Connection struct {
	server   *http.Server
	host     string
	port     int
	messages chan string
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
	conn = &Connection{}
	conn.host = host
	conn.port = port
	conn.server = &http.Server{
		Addr:           address,
		Handler:        conn,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	connections[address] = conn
	close(conn.messages)
	return conn, nil
}

func (conn *Connection) GetMessage() (string, error) {
	msg, isOpenChan := <-conn.messages
	if !isOpenChan {
		return "", errors.New("connection channel is closed")
	}
	return msg, nil
}

func (conn *Connection) Open() error {
	listener, err := net.Listen("tcp", conn.server.Addr)
	if err != nil {
		return err
	}
	go func() {
		conn.server.Serve(listener)
	}()
	res, err := http.Get(fmt.Sprintf("http://localhost:%d", conn.port))
	if err != nil {
		return err
	}
	err = res.Body.Close()
	if err != nil {
		return err
	}
	conn.messages = make(chan string)
	fmt.Printf("server is listening on port %d\n", conn.port)
	return nil
}

func (conn *Connection) Close() {
	close(conn.messages)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()
	err := conn.server.Shutdown(shutdownCtx)
	if err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")
}

func (conn *Connection) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	msg, err := conn.GetMessage()
	if err == nil {
		conn.messages <- msg
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(request.Body)
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
