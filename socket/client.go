package socket

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	addr      string
	clientID  uuid.UUID
	channelID uuid.UUID
	ctx       *connctx
	done      chan struct{}
	connMu    sync.Mutex
}

func NewClient(addr string, channelID uuid.UUID) (*Client, error) {
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
		done:      make(chan struct{}),
		ctx: &connctx{
			conn:   conn,
			writes: make(chan *frame, 1024),
			reads:  make(chan *frame, 1024),
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

func (c *Client) sendStartFrame(reqID uuid.UUID, timeoutMs uint64, flags uint16) error {
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.RequestID = reqID
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flags | flagRequest

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return errClosed
	}

	return c.awaitFrame(reqID, byte(startFrame))
}

func (c *Client) sendChunkFrame(reqID uuid.UUID, payload []byte) ([]byte, error) {
	f := getFrame()
	f.Version = version1
	f.Type = chunkFrame
	f.RequestID = reqID
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.Flags = flagRequest
	f.Payload = make([]byte, len(payload))
	copy(f.Payload, payload)

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return nil, errClosed
	}

	return c.awaitPayload(reqID, byte(chunkFrame))
}

func (c *Client) sendEndFrame(reqID uuid.UUID, timeoutMs uint64) ([]byte, error) {
	f := getFrame()
	f.Version = version1
	f.Type = endFrame
	f.RequestID = reqID
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flagRequest

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return nil, errClosed
	}

	return c.awaitPayload(reqID, byte(endFrame))
}

func (c *Client) Send(r io.Reader, timeout time.Duration) (io.Reader, error) {
	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	// Send start frame
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.RequestID = reqID
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flagRequest

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return nil, errClosed
	}

	var respBuf bytes.Buffer
	buf := getBuffer()
	defer putBuffer(buf)

	for {
		n, err := r.Read(*buf)
		if n > 0 {
			payload, err := c.sendChunkFrame(reqID, (*buf)[:n])
			if err != nil {
				return nil, err
			}
			respBuf.Write(payload)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	payload, err := c.sendEndFrame(reqID, timeoutMs)
	if err != nil {
		return nil, err
	}
	respBuf.Write(payload)

	return bytes.NewReader(respBuf.Bytes()), nil
}

func (c *Client) respondStartFrame(timeoutMs uint64) (uuid.UUID, uuid.UUID, uint16, error) {
	select {
	case f := <-c.ctx.reads:
		if f.Type != startFrame {
			putFrame(f)
			return uuid.Nil, uuid.Nil, 0, errPeerError
		}
		reqID := f.RequestID
		requesterID := f.ClientID
		flags := f.Flags
		putFrame(f)

		resp := getFrame()
		resp.Version = version1
		resp.Type = startFrame
		resp.RequestID = reqID
		resp.ClientID = requesterID
		resp.ChannelID = c.channelID
		resp.ClientTimeoutMs = timeoutMs
		resp.Flags = flagResponse

		select {
		case c.ctx.writes <- resp:
			return reqID, requesterID, flags, nil
		case <-c.done:
			putFrame(resp)
			return uuid.Nil, uuid.Nil, 0, errClosed
		}
	case <-time.After(5 * time.Second):
		return uuid.Nil, uuid.Nil, 0, errTimeout
	case <-c.done:
		return uuid.Nil, uuid.Nil, 0, errClosed
	}
}

func (c *Client) respondChunkFrame(reqID, requesterID uuid.UUID, r io.Reader, buf *[]byte) error {
	select {
	case f := <-c.ctx.reads:
		if f.RequestID != reqID {
			putFrame(f)
			return errPeerError
		}
		if f.Type == endFrame {
			putFrame(f)
			return io.EOF
		}
		if f.Type != chunkFrame {
			putFrame(f)
			return errPeerError
		}
		putFrame(f)
	case <-time.After(5 * time.Second):
		return errTimeout
	case <-c.done:
		return errClosed
	}

	n, err := r.Read(*buf)
	resp := getFrame()
	resp.Version = version1
	resp.Type = chunkFrame
	resp.RequestID = reqID
	resp.ClientID = requesterID
	resp.ChannelID = c.channelID
	resp.Flags = flagResponse
	if n > 0 {
		resp.Payload = make([]byte, n)
		copy(resp.Payload, (*buf)[:n])
	}

	select {
	case c.ctx.writes <- resp:
	case <-c.done:
		putFrame(resp)
		return errClosed
	}
	return err
}

func (c *Client) Respond(r io.Reader, timeout time.Duration) error {
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	// Receive start frame
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.Flags = flagReceive

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return errClosed
	}

	// Wait for start frame
	var reqID, requesterID uuid.UUID
	select {
	case sf := <-c.ctx.reads:
		if sf.Type != startFrame {
			putFrame(sf)
			return errPeerError
		}
		reqID = sf.RequestID
		requesterID = sf.ClientID
		putFrame(sf)
	case <-time.After(5 * time.Second):
		return errTimeout
	case <-c.done:
		return errClosed
	}

	// Send start response
	resp := getFrame()
	resp.Version = version1
	resp.Type = startFrame
	resp.RequestID = reqID
	resp.ClientID = requesterID
	resp.ChannelID = c.channelID
	resp.ClientTimeoutMs = timeoutMs
	resp.Flags = flagResponse

	select {
	case c.ctx.writes <- resp:
	case <-c.done:
		putFrame(resp)
		return errClosed
	}

	buf := getBuffer()
	defer putBuffer(buf)

	for {
		err := c.respondChunkFrame(reqID, requesterID, r, buf)
		if err == io.EOF {
			resp := getFrame()
			resp.Version = version1
			resp.Type = endFrame
			resp.RequestID = reqID
			resp.ClientID = requesterID
			resp.ChannelID = c.channelID
			resp.ClientTimeoutMs = timeoutMs
			resp.Flags = flagResponse
			select {
			case c.ctx.writes <- resp:
				return nil
			case <-c.done:
				putFrame(resp)
				return errClosed
			}
		}
		if err != nil {
			return err
		}
	}
}

func (c *Client) Receive() (*frame, error) {
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.Flags = flagReceive

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return nil, errClosed
	}

	select {
	case resp := <-c.ctx.reads:
		return resp, nil
	case <-c.done:
		return nil, errClosed
	}
}

func (c *Client) awaitFrame(reqID uuid.UUID, expectedType byte) error {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID {
				if f.Type == errorFrame {
					putFrame(f)
					return errPeerError
				}
				if f.Type == expectedType {
					putFrame(f)
					return nil
				}
			}
			putFrame(f)
		case <-c.done:
			return errClosed
		}
	}
}

func (c *Client) awaitPayload(reqID uuid.UUID, expectedType byte) ([]byte, error) {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID {
				if f.Type == errorFrame {
					putFrame(f)
					return nil, errPeerError
				}
				if f.Type == expectedType {
					payload := f.Payload
					putFrame(f)
					return payload, nil
				}
			}
			putFrame(f)
		case <-c.done:
			return nil, errClosed
		}
	}
}

func (c *Client) Close() error {
	close(c.done)

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.ctx != nil {
		if c.ctx.conn != nil {
			c.ctx.conn.Close()
		}
		close(c.ctx.writes)
		close(c.ctx.errors)
		c.ctx.wg.Wait()
		c.ctx.conn = nil
	}
	return nil
}
