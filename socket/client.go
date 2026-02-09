package socket

import (
	"bytes"
	"io"
	"log"
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
			handshake: make(chan struct{}),
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

	c.ctx.wg.Add(3)
	go c.ctx.startWriter()
	go c.ctx.startReader()

	go func() {
		defer c.ctx.wg.Done()
		select {
		case <-c.done:
		case <-c.paired:
		}
	}()

	return c, nil
}

func (c *Client) sendStartFrame(reqID uuid.UUID, timeoutMs uint64, frameSeq int, flags uint16) error {

	startF := getFrame()
	startF.Version = version1
	startF.Type = startFrame
	startF.RequestID = reqID
	startF.ClientID = c.clientID
	startF.PeerClientID = c.peerID
	startF.ChannelID = c.channelID
	startF.ClientTimeoutMs = timeoutMs
	startF.Flags = flags
	log.Printf("[Client %s] Sending START frame [seq=%d] for request %s", c.clientID, frameSeq, reqID)
	select {
	case c.ctx.writes <- startF:
	case <-c.done:
		putFrame(startF)
		return errClosed
	}

	for {
		select {
		case f := <-c.ctx.reads:
			switch f.Type {
			case errorFrame:
				log.Printf("[Client %s] ERROR: Received error frame during start frame response", c.clientID)
				putFrame(f)
				return errPeerError
			case startFrame:
				if f.RequestID == reqID {
					log.Printf("[Client %s] Received start frame response for request %s", c.clientID, reqID)
					if c.peerID != f.ClientID {
						log.Printf("[Client %s] ERROR: Peer ID mismatch, expected %s, got %s", c.clientID, c.peerID, f.ClientID)
						putFrame(f)
						return errPeerError
					}
					putFrame(f)
					return nil
				}
				putFrame(f)
			}
		case <-c.done:
			return errClosed
		}
	}
}

func (c *Client) sendChunkFrame(reqID uuid.UUID, payload []byte, frameSeq int) (io.Reader, error) {
	chunk := getFrame()
	chunk.Version = version1
	chunk.Type = chunkFrame
	chunk.RequestID = reqID
	chunk.ClientID = c.clientID
	chunk.PeerClientID = c.peerID
	chunk.ChannelID = c.channelID
	chunk.Payload = make([]byte, len(payload))
	copy(chunk.Payload, payload)
	log.Printf("[Client %s] Sending CHUNK frame [seq=%d] for request %s (size=%d)", c.clientID, frameSeq, reqID, len(payload))
	select {
	case c.ctx.writes <- chunk:
	case <-c.done:
		putFrame(chunk)
		return nil, errClosed
	}

	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID {
				if f.Type == errorFrame {
					log.Printf("[Client %s] ERROR: Received error frame after CHUNK: %s", c.clientID, string(f.Payload))
					putFrame(f)
					return nil, errPeerError
				}
				if f.Type == chunkFrame {
					log.Printf("[Client %s] Received CHUNK frame response for request %s", c.clientID, reqID)
					r := bytes.NewReader(f.Payload)
					putFrame(f)
					return r, nil
				}
			}
			putFrame(f)
		case <-c.done:
			return nil, errClosed
		}
	}
}

func (c *Client) sendEndFrame(reqID uuid.UUID, timeoutMs uint64, frameSeq int) (io.Reader, error) {
	endF := getFrame()
	endF.Version = version1
	endF.Type = endFrame
	endF.RequestID = reqID
	endF.ClientID = c.clientID
	endF.PeerClientID = c.peerID
	endF.ChannelID = c.channelID
	endF.ClientTimeoutMs = timeoutMs
	log.Printf("[Client %s] Sending END frame [seq=%d] for request %s", c.clientID, frameSeq, reqID)
	select {
	case c.ctx.writes <- endF:
	case <-c.done:
		putFrame(endF)
		return nil, errClosed
	}

	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID {
				if f.Type == errorFrame {
					log.Printf("[Client %s] ERROR: Received error frame after END: %s", c.clientID, string(f.Payload))
					putFrame(f)
					return nil, errPeerError
				}
				if f.Type == endFrame {
					log.Printf("[Client %s] Received END frame response for request %s", c.clientID, reqID)
					r := bytes.NewReader(f.Payload)
					putFrame(f)
					return r, nil
				}
			}
			putFrame(f)
		case <-c.done:
			return nil, errClosed
		}
	}
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

	frameSeq := 0

	if err := c.sendStartFrame(reqID, timeoutMs, frameSeq, flagNone); err != nil {
		return nil, err
	}
	frameSeq++

	respBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf := getBuffer()
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			chunkResp, err := c.sendChunkFrame(reqID, (*buf)[:n], frameSeq)
			if err != nil {
				putBuffer(buf)
				return nil, err
			}
			io.Copy(respBuf, chunkResp)
			frameSeq++
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

	endResp, err := c.sendEndFrame(reqID, timeoutMs, frameSeq)
	if err != nil {
		return nil, err
	}
	io.Copy(respBuf, endResp)

	log.Printf("[Client %s] Received complete response for request %s", c.clientID, reqID)
	return bytes.NewReader(respBuf.Bytes()), nil
}

func (c *Client) respondStartFrame(timeoutMs uint64) (uuid.UUID, error) {
	var reqID uuid.UUID
	select {
	case f := <-c.ctx.reads:
		if f.Type != startFrame {
			putFrame(f)
			return uuid.Nil, errPeerError
		}
		reqID = f.RequestID
		flags := f.Flags
		putFrame(f)

		startF := getFrame()
		startF.Version = version1
		startF.Type = startFrame
		startF.RequestID = reqID
		startF.ClientID = c.clientID
		startF.PeerClientID = c.peerID
		startF.ChannelID = c.channelID
		startF.ClientTimeoutMs = timeoutMs
		startF.Flags = flags
		select {
		case c.ctx.writes <- startF:
		case <-c.done:
			putFrame(startF)
			return uuid.Nil, errClosed
		}
		return reqID, nil
	case <-c.done:
		return uuid.Nil, errClosed
	}
}

func (c *Client) respondChunkFrame(reqID uuid.UUID, r io.Reader, buf *[]byte) error {
	select {
	case f := <-c.ctx.reads:
		if f.RequestID != reqID {
			putFrame(f)
			return nil
		}
		if f.Type != chunkFrame {
			putFrame(f)
			return errPeerError
		}
		putFrame(f)
	case <-c.done:
		return errClosed
	}

	n, err := r.Read(*buf)
	if n > 0 {
		chunk := getFrame()
		chunk.Version = version1
		chunk.Type = chunkFrame
		chunk.RequestID = reqID
		chunk.ClientID = c.clientID
		chunk.PeerClientID = c.peerID
		chunk.ChannelID = c.channelID
		chunk.Payload = make([]byte, n)
		copy(chunk.Payload, (*buf)[:n])
		select {
		case c.ctx.writes <- chunk:
		case <-c.done:
			putFrame(chunk)
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

	endF := getFrame()
	endF.Version = version1
	endF.Type = endFrame
	endF.RequestID = reqID
	endF.ClientID = c.clientID
	endF.PeerClientID = c.peerID
	endF.ChannelID = c.channelID
	endF.ClientTimeoutMs = timeoutMs
	select {
	case c.ctx.writes <- endF:
	case <-c.done:
		putFrame(endF)
		return errClosed
	}
	return nil
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
