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

func (c *Client) sendStartFrame(reqID uuid.UUID, clientID uuid.UUID, timeoutMs uint64, flags uint16) error {
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.RequestID = reqID
	f.ClientID = clientID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flags
	f.setSequence(0, true)
	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return errClosed
	}
	return nil
}

func (c *Client) receiveStartFrame(reqID uuid.UUID) (*frame, error) {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID || reqID == uuid.Nil {
				if f.Type == errorFrame {
					putFrame(f)
					return nil, errFrame
				}
				if f.Type != startFrame {
					putFrame(f)
					return nil, errInvalidFrame
				}
				putFrame(f)
				return f, nil
			}
			putFrame(f)
		case <-c.done:
			return nil, errClosed
		}
	}
}

func (c *Client) sendChunkFrame(reqID uuid.UUID, clientID uuid.UUID, timeoutMs uint64, index int, isFinal bool, payload []byte, flags uint16) error {
	f := getFrame()
	f.Version = version1
	f.Type = chunkFrame
	f.RequestID = reqID
	f.ClientID = clientID
	f.ChannelID = c.channelID
	f.Flags = flags
	f.ClientTimeoutMs = timeoutMs
	f.Payload = make([]byte, len(payload))
	copy(f.Payload, payload)
	f.setSequence(index, isFinal)
	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return errClosed
	}
	return nil
}

func (c *Client) receiveChunkFrame(reqID uuid.UUID) (*frame, error) {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID || reqID == uuid.Nil {
				if f.Type == errorFrame {
					putFrame(f)
					return nil, errFrame
				}
				if f.Type != chunkFrame {
					putFrame(f)
					return nil, errInvalidFrame
				}
				putFrame(f)
				return f, nil
			}
			putFrame(f)
		case <-c.done:
			return nil, errClosed
		}
	}
}

func (c *Client) sendEndFrame(reqID uuid.UUID, clientID uuid.UUID, timeoutMs uint64, flags uint16) error {
	f := getFrame()
	f.Version = version1
	f.Type = endFrame
	f.RequestID = reqID
	f.ClientID = clientID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flags
	f.setSequence(0, true)
	select {
	case c.ctx.writes <- f:
	case <-c.done:
		putFrame(f)
		return errClosed
	}
	return nil
}

func (c *Client) receiveEndFrame(reqID uuid.UUID) (*frame, error) {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID || reqID == uuid.Nil {
				if f.Type == errorFrame {
					putFrame(f)
					return nil, errFrame
				}
				if f.Type != endFrame {
					putFrame(f)
					return nil, errInvalidFrame
				}
				putFrame(f)
				return f, nil
			}
			putFrame(f)
		case <-c.done:
			return nil, errClosed
		}
	}
}

func (c *Client) sendDisassembledChunkFrames(reqId uuid.UUID, clientID uuid.UUID, r io.Reader, timeoutMs uint64, flags uint16) error {
	buf := getBuffer()
	defer putBuffer(buf)

	index := 0
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			isFinal := err == io.EOF
			if err := c.sendChunkFrame(reqId, clientID, timeoutMs, index, isFinal, (*buf)[:n], flags); err != nil {
				return err
			}
			index++
			if isFinal {
				break
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) receiveAssembledChunkFrames(reqId uuid.UUID, clientId uuid.UUID, timeoutMs uint64, flags uint16) (buff bytes.Buffer, requestId uuid.UUID, clientID uuid.UUID, err error) {
	for {
		var rcvF *frame
		if flags&flagReceive != 0 {
			if err = c.sendChunkFrame(reqId, clientId, timeoutMs, 0, true, nil, flags); err != nil {
				return
			}
			rcvF, err = c.receiveChunkFrame(uuid.Nil)
		} else {
			rcvF, err = c.receiveChunkFrame(reqId)
		}
		if err != nil {
			return
		}
		requestId = rcvF.RequestID
		clientID = rcvF.ClientID
		buff.Write(rcvF.Payload)
		if rcvF.isFinal() {
			break
		}
	}
	return
}

func (c *Client) Send(r io.Reader, timeout time.Duration) (io.Reader, error) {
	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	if err := c.sendStartFrame(reqID, c.clientID, timeoutMs, flagRequest); err != nil {
		return nil, err
	}

	if _, err := c.receiveStartFrame(reqID); err != nil {
		return nil, err
	}

	if err := c.sendDisassembledChunkFrames(reqID, c.clientID, r, timeoutMs, flagRequest); err != nil {
		return nil, err
	}

	respBuf, _, _, err := c.receiveAssembledChunkFrames(reqID, c.clientID, timeoutMs, flagRequest)
	if err != nil {
		return nil, err
	}

	if err := c.sendEndFrame(reqID, c.clientID, timeoutMs, flagRequest); err != nil {
		return nil, err
	}

	if _, err := c.receiveEndFrame(reqID); err != nil {
		return nil, err
	}

	return bytes.NewReader(respBuf.Bytes()), nil
}

func (c *Client) Receive(incoming chan io.Reader, outgoing chan io.Reader, timeout time.Duration) error {
	origReqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	if err := c.sendStartFrame(origReqID, c.clientID, timeoutMs, flagReceive); err != nil {
		return err
	}

	rcvF, err := c.receiveStartFrame(uuid.Nil)
	if err != nil {
		return err
	}

	reqID := rcvF.RequestID
	reqClientID := rcvF.ClientID

	if err := c.sendStartFrame(reqID, reqClientID, timeoutMs, flagResponse); err != nil {
		return err
	}

	reqBuf, _, _, err := c.receiveAssembledChunkFrames(origReqID, c.clientID, timeoutMs, flagReceive)
	if err != nil {
		return err
	}

	incoming <- bytes.NewReader(reqBuf.Bytes())
	reader := <-outgoing

	if err := c.sendDisassembledChunkFrames(reqID, reqClientID, reader, timeoutMs, flagResponse); err != nil {
		return err
	}

	if err := c.sendEndFrame(origReqID, c.clientID, timeoutMs, flagReceive); err != nil {
		return err
	}

	if _, err := c.receiveEndFrame(uuid.Nil); err != nil {
		return err
	}
	return nil
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
