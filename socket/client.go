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
	handshakeClient
	addr   string
	connMu sync.Mutex
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
		addr: addr,
		handshakeClient: handshakeClient{
			clientID:  uuid.New(),
			peerID:    uuid.Nil,
			channelID: channelID,
			done:      make(chan struct{}),
			paired:    make(chan struct{}),
			ctx: &connctx{
				conn:   conn,
				writes: make(chan *frame, 1024),
				reads:  make(chan *frame, 100),
				errors: make(chan *frame, 256),
				closed: make(chan struct{}),
				wg:     &sync.WaitGroup{},
			},
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
	f.PeerClientID = c.peerID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flags

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
	f.PeerClientID = c.peerID
	f.ChannelID = c.channelID
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
	f.PeerClientID = c.peerID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs

	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return nil, errClosed
	}

	return c.awaitPayload(reqID, byte(endFrame))
}

func (c *Client) Send(r io.Reader, timeout time.Duration) (io.Reader, error) {
	if err := c.doHandshake(); err != nil {
		return nil, err
	}

	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	if err := c.sendStartFrame(reqID, timeoutMs, flagNone); err != nil {
		return nil, err
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

func (c *Client) respondStartFrame(timeoutMs uint64) (uuid.UUID, error) {
	select {
	case f := <-c.ctx.reads:
		if f.Type != startFrame {
			putFrame(f)
			return uuid.Nil, errPeerError
		}
		reqID := f.RequestID
		flags := f.Flags
		putFrame(f)

		resp := getFrame()
		resp.Version = version1
		resp.Type = startFrame
		resp.RequestID = reqID
		resp.ClientID = c.clientID
		resp.PeerClientID = c.peerID
		resp.ChannelID = c.channelID
		resp.ClientTimeoutMs = timeoutMs
		resp.Flags = flags

		select {
		case c.ctx.writes <- resp:
			return reqID, nil
		case <-c.done:
			putFrame(resp)
			return uuid.Nil, errClosed
		}
	case <-c.done:
		return uuid.Nil, errClosed
	}
}

func (c *Client) respondChunkFrame(reqID uuid.UUID, r io.Reader, buf *[]byte) error {
	select {
	case f := <-c.ctx.reads:
		if f.RequestID != reqID || f.Type != chunkFrame {
			putFrame(f)
			return errPeerError
		}
		putFrame(f)
	case <-c.done:
		return errClosed
	}

	n, err := r.Read(*buf)
	if n > 0 {
		resp := getFrame()
		resp.Version = version1
		resp.Type = chunkFrame
		resp.RequestID = reqID
		resp.ClientID = c.clientID
		resp.PeerClientID = c.peerID
		resp.ChannelID = c.channelID
		resp.Payload = make([]byte, n)
		copy(resp.Payload, (*buf)[:n])

		select {
		case c.ctx.writes <- resp:
		case <-c.done:
			putFrame(resp)
			return errClosed
		}
	}
	return err
}

func (c *Client) respondEndFrame(reqID uuid.UUID, timeoutMs uint64) error {
	select {
	case f := <-c.ctx.reads:
		if f.RequestID != reqID || f.Type != endFrame {
			putFrame(f)
			return errPeerError
		}
		putFrame(f)
	case <-c.done:
		return errClosed
	}

	resp := getFrame()
	resp.Version = version1
	resp.Type = endFrame
	resp.RequestID = reqID
	resp.ClientID = c.clientID
	resp.PeerClientID = c.peerID
	resp.ChannelID = c.channelID
	resp.ClientTimeoutMs = timeoutMs

	select {
	case c.ctx.writes <- resp:
		return nil
	case <-c.done:
		putFrame(resp)
		return errClosed
	}
}

func (c *Client) Respond(r io.Reader, timeout time.Duration) error {
	if err := c.doHandshake(); err != nil {
		return err
	}
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	reqID, err := c.respondStartFrame(timeoutMs)
	if err != nil {
		return err
	}

	buf := getBuffer()
	defer putBuffer(buf)

	for {
		err := c.respondChunkFrame(reqID, r, buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return c.respondEndFrame(reqID, timeoutMs)
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
					if f.ClientID != c.peerID {
						putFrame(f)
						return errPeerError
					}
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
