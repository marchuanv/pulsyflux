package socket

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	addr      string
	ctx       *connctx
	connMu    sync.Mutex
	clientID  uuid.UUID
	peerID    uuid.UUID
	channelID uuid.UUID
	role      clientRole
	done      chan struct{}
}

func NewClient(addr string, channelID uuid.UUID, role clientRole) (*Client, error) {
	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		return nil, err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetReadBuffer(2 * 1024 * 1024)
		tcpConn.SetWriteBuffer(2 * 1024 * 1024)
	}

	c := &Client{
		addr:      addr,
		clientID:  uuid.New(),
		channelID: channelID,
		role:      role,
		done:      make(chan struct{}),
		ctx: &connctx{
			conn:   conn,
			writes: make(chan *frame, 1024),
			reads:  make(chan *frame, 100),
			errors: make(chan *frame, 256),
			closed: make(chan struct{}),
			wg:     &sync.WaitGroup{},
		},
	}

	c.ctx.wg.Add(2)
	go c.ctx.startWriter()
	go c.ctx.startReader()

	return c, nil
}

// Respond sends a response to an incoming request using the same request ID
func (c *Client) Respond(reqID uuid.UUID, r io.Reader, timeout time.Duration) error {
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	peerID := c.peerID
	frameSeq := 0

	startF := getFrame()
	startF.Version = version1
	startF.Type = startFrame
	startF.RequestID = reqID
	startF.ClientID = c.clientID
	startF.PeerClientID = peerID
	startF.ClientTimeoutMs = timeoutMs
	log.Printf("[Client %s] Responding with START frame [seq=%d] for request %s", c.clientID, frameSeq, reqID)
	frameSeq++
	select {
	case c.ctx.writes <- startF:
	case <-c.done:
		putFrame(startF)
		return errClosed
	}

	buf := getBuffer()
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			chunk := getFrame()
			chunk.Version = version1
			chunk.Type = chunkFrame
			chunk.RequestID = reqID
			chunk.ClientID = c.clientID
			chunk.PeerClientID = peerID
			chunk.Payload = make([]byte, n)
			copy(chunk.Payload, (*buf)[:n])
			log.Printf("[Client %s] Responding with CHUNK frame [seq=%d] for request %s (size=%d)", c.clientID, frameSeq, reqID, n)
			frameSeq++
			select {
			case c.ctx.writes <- chunk:
			case <-c.done:
				putBuffer(buf)
				putFrame(chunk)
				return errClosed
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			putBuffer(buf)
			return err
		}
	}
	putBuffer(buf)

	endF := getFrame()
	endF.Version = version1
	endF.Type = endFrame
	endF.RequestID = reqID
	endF.ClientID = c.clientID
	endF.PeerClientID = peerID
	endF.ClientTimeoutMs = timeoutMs
	log.Printf("[Client %s] Responding with END frame [seq=%d] for request %s", c.clientID, frameSeq, reqID)
	frameSeq++
	select {
	case c.ctx.writes <- endF:
	case <-c.done:
		putFrame(endF)
		return errClosed
	}

	return nil
}

func (c *Client) Send(r io.Reader, timeout time.Duration) (io.Reader, error) {
	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	peerID := c.peerID

	// Track frame sequence for this request
	frameSeq := 0

	startF := getFrame()
	startF.Version = version1
	startF.Type = startFrame
	startF.RequestID = reqID
	startF.ClientID = c.clientID
	startF.PeerClientID = peerID
	startF.ClientTimeoutMs = timeoutMs
	log.Printf("[Client %s] Sending START frame [seq=%d] for request %s", c.clientID, frameSeq, reqID)
	frameSeq++
	select {
	case c.ctx.writes <- startF:
	case <-c.done:
		putFrame(startF)
		return nil, errClosed
	}

	buf := getBuffer()
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			chunk := getFrame()
			chunk.Version = version1
			chunk.Type = chunkFrame
			chunk.RequestID = reqID
			chunk.ClientID = c.clientID
			chunk.PeerClientID = peerID
			chunk.Payload = make([]byte, n)
			copy(chunk.Payload, (*buf)[:n])
			log.Printf("[Client %s] Sending CHUNK frame [seq=%d] for request %s (size=%d)", c.clientID, frameSeq, reqID, n)
			frameSeq++
			select {
			case c.ctx.writes <- chunk:
			case <-c.done:
				putBuffer(buf)
				putFrame(chunk)
				return nil, errClosed
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			putBuffer(buf)
			return nil, err
		}
	}
	putBuffer(buf)

	endF := getFrame()
	endF.Version = version1
	endF.Type = endFrame
	endF.RequestID = reqID
	endF.ClientID = c.clientID
	endF.PeerClientID = peerID
	endF.ClientTimeoutMs = timeoutMs
	log.Printf("[Client %s] Sending END frame [seq=%d] for request %s", c.clientID, frameSeq, reqID)
	frameSeq++
	select {
	case c.ctx.writes <- endF:
	case <-c.done:
		putFrame(endF)
		return nil, errClosed
	}

	// Receive and assemble response
	deadline := time.After(timeout)
	respBuf := getBuffer()
	payload := (*respBuf)[:0]
	gotStart := false

	for {
		select {
		case <-deadline:
			putBuffer(respBuf)
			return nil, errTimeout
		case f := <-c.ctx.reads:
			if f.RequestID != reqID {
				putFrame(f)
				continue
			}

			switch f.Type {
			case startFrame:
				gotStart = true
				putFrame(f)
			case chunkFrame:
				payload = append(payload, f.Payload...)
				putFrame(f)
			case endFrame:
				if !gotStart {
					putBuffer(respBuf)
					putFrame(f)
					log.Printf("[Client %s] ERROR: Received END before START for request %s", c.clientID, reqID)
					return nil, errors.New("received end before start")
				}
				log.Printf("[Client %s] Received complete response for request %s (size=%d)", c.clientID, reqID, len(payload))
				result := make([]byte, len(payload))
				copy(result, payload)
				putBuffer(respBuf)
				putFrame(f)
				return bytes.NewReader(result), nil
			case errorFrame:
				putBuffer(respBuf)
				putFrame(f)
				return nil, errPeerError
			}
		case <-c.done:
			putBuffer(respBuf)
			return nil, errClosed
		}
	}
}

func (c *Client) Receive() (uuid.UUID, io.Reader, bool) {
	streamReqs := make(map[uuid.UUID]*bytes.Buffer)

	for {
		select {
		case f, ok := <-c.ctx.reads:
			if !ok {
				return uuid.Nil, nil, false
			}

			switch f.Type {
			case startFrame:
				streamReqs[f.RequestID] = bytes.NewBuffer(make([]byte, 0, 1024))
				putFrame(f)

			case chunkFrame:
				buf := streamReqs[f.RequestID]
				if buf != nil {
					buf.Write(f.Payload)
				}
				putFrame(f)

			case endFrame:
				buf := streamReqs[f.RequestID]
				delete(streamReqs, f.RequestID)
				reqID := f.RequestID
				putFrame(f)
				if buf != nil {
					return reqID, bytes.NewReader(buf.Bytes()), true
				}

			case errorFrame:
				putFrame(f)
			}

		case <-c.done:
			return uuid.Nil, nil, false
		}
	}
}

func (c *Client) Close() error {
	log.Printf("[Client %s] Closing client", c.clientID)
	close(c.done)

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.ctx != nil {
		close(c.ctx.writes)
		close(c.ctx.errors)
		c.ctx.wg.Wait()
		if c.ctx.conn != nil {
			err := c.ctx.conn.Close()
			c.ctx.conn = nil
			return err
		}
	}
	return nil
}
