package socket

import (
	"bytes"
	"errors"
	"io"
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
	incoming  chan *frame
	wg        sync.WaitGroup
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
		incoming:  make(chan *frame, 100),
		ctx: &connctx{
			conn:   conn,
			writes: make(chan *frame, 1024),
			errors: make(chan *frame, 256),
			closed: make(chan struct{}),
			wg:     &sync.WaitGroup{},
		},
	}

	c.ctx.wg.Add(1)
	go c.ctx.startWriter()

	c.wg.Add(1)
	go c.readLoop()

	// Perform handshake
	if err := c.handshake(); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) handshake() error {
	// Start responder goroutine
	c.wg.Add(1)
	go c.handshakeResponder()

	// Build handshake payload: role (1 byte) + channelID (16 bytes)
	payload := make([]byte, 17)
	payload[0] = byte(c.role)
	copy(payload[1:17], c.channelID[:])

	// Send handshake using normal Send()
	resp, err := c.Send(bytes.NewReader(payload), 5*time.Second)
	if err != nil {
		return err
	}

	// Read and validate response
	peerPayload, err := io.ReadAll(resp)
	if err != nil || len(peerPayload) < 17 {
		return errors.New("invalid handshake response")
	}

	peerRole := clientRole(peerPayload[0])
	var peerChannelID uuid.UUID
	copy(peerChannelID[:], peerPayload[1:17])

	if peerChannelID != c.channelID {
		return errors.New("channel ID mismatch")
	}
	if peerRole == c.role {
		return errors.New("role conflict")
	}

	return nil
}

func (c *Client) handshakeResponder() {
	defer c.wg.Done()
	_, r, ok := c.Receive()
	if !ok {
		return
	}

	payload, err := io.ReadAll(r)
	if err != nil || len(payload) < 17 {
		return
	}

	peerRole := clientRole(payload[0])
	var peerChannelID uuid.UUID
	copy(peerChannelID[:], payload[1:17])

	if peerChannelID != c.channelID || peerRole == c.role {
		return
	}

	// Build and send response
	response := make([]byte, 17)
	response[0] = byte(c.role)
	copy(response[1:17], c.channelID[:])
	c.Send(bytes.NewReader(response), 5*time.Second)
}

func (c *Client) Send(r io.Reader, timeout time.Duration) (io.Reader, error) {
	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	// Send start frame
	startF := getFrame()
	startF.Version = version1
	startF.Type = startFrame
	startF.RequestID = reqID
	startF.ClientID = c.clientID
	startF.PeerClientID = c.peerID
	startF.ClientTimeoutMs = timeoutMs
	if !c.ctx.enqueue(startF) {
		putFrame(startF)
		return nil, errors.New("failed to enqueue start frame")
	}

	// Send chunks
	buf := getBuffer()
	chunk := getFrame()
	chunk.Version = version1
	chunk.Type = chunkFrame
	chunk.RequestID = reqID
	chunk.ClientID = c.clientID
	chunk.PeerClientID = c.peerID
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			chunk.Payload = (*buf)[:n]
			if !c.ctx.enqueue(chunk) {
				putBuffer(buf)
				putFrame(chunk)
				return nil, errors.New("failed to enqueue chunk")
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			putBuffer(buf)
			putFrame(chunk)
			return nil, err
		}
	}
	putBuffer(buf)
	putFrame(chunk)

	// Send end frame
	endF := getFrame()
	endF.Version = version1
	endF.Type = endFrame
	endF.RequestID = reqID
	endF.ClientID = c.clientID
	endF.PeerClientID = c.peerID
	endF.ClientTimeoutMs = timeoutMs
	if !c.ctx.enqueue(endF) {
		putFrame(endF)
		return nil, errors.New("failed to enqueue end frame")
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
		case f := <-c.incoming:
			if f.RequestID != reqID {
				select {
				case c.incoming <- f:
				default:
					putFrame(f)
				}
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
					return nil, errors.New("received end before start")
				}
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

func (c *Client) readLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		default:
		}

		f, err := c.ctx.readFrame()
		if err != nil {
			return
		}

		select {
		case c.incoming <- f:
		case <-c.done:
			putFrame(f)
			return
		}
	}
}

func (c *Client) Receive() (uuid.UUID, io.Reader, bool) {
	streamReqs := make(map[uuid.UUID]*bytes.Buffer)

	for {
		select {
		case f, ok := <-c.incoming:
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
	close(c.done)
	c.wg.Wait()
	close(c.incoming)

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
