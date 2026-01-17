package socket

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Server struct {
	port   string
	closed bool
	ln     net.Listener
}

func NewServer(port string) *Server {
	return &Server{port, false, nil}
}

func (s *Server) Start() {
	ln, err := net.Listen("tcp", "localhost:"+s.port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	s.ln = ln
	s.closed = false
	go func() {
		for s.closed == false {
			conn, err := s.ln.Accept()
			if err != nil {
				continue
			}
			handle(conn)
		}
	}()
}

func (s *Server) Stop() {
	if !s.closed {
		s.ln.Close()
		s.ln = nil
		s.closed = true
	}
}

func handle(conn net.Conn) {
	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

	for {
		// 1. READ: Set a deadline to prevent hanging (e.g., 2 minutes)
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

		// Read message until newline
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("client closed connection.")
			} else {
				fmt.Println("Read error:", err)
			}
			return
		}

		// Clean up the input (remove \r\n)
		input := strings.TrimSpace(message)
		fmt.Printf("Received: %s\n", input)

		// 2. PROCESS: Prepare a response
		response := "ACK: " + input + "\n"

		// 3. WRITE: Send the response back to the client
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Write error:", err)
			return
		}
	}
}
