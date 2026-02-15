package socket

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Client struct {
	addr      string
	clientID  uuid.UUID
	channelID uuid.UUID
	ctx       *connctx
	done      chan struct{}
	connMu    sync.Mutex
	opMu      sync.Mutex
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

	// Send registration frame to trigger server-side registration
	if err := c.sendRegistration(); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) sendRegistration() error {
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.Flags = flagNone
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.RequestID = uuid.Nil
	select {
	case c.ctx.writes <- f:
		return nil
	case <-c.ctx.closed:
		putFrame(f)
		return errClosed
	}
}

func (c *Client) sendFrame(frameType byte, reqID uuid.UUID, timeoutMs uint64, flags uint16, payload []byte, index int, isFinal bool) error {
	select {
	case <-c.ctx.closed:
		return errClosed
	default:
	}
	
	f := getFrame()
	f.Version = version1
	f.Type = frameType
	f.RequestID = reqID
	f.ClientID = c.clientID
	f.ChannelID = c.channelID
	f.ClientTimeoutMs = timeoutMs
	f.Flags = flags
	if payload != nil {
		f.Payload = make([]byte, len(payload))
		copy(f.Payload, payload)
	}
	if frameType == chunkFrame {
		f.setSequence(index, isFinal)
	} else {
		f.setSequence(0, true)
	}
	select {
	case c.ctx.writes <- f:
		return nil
	case <-c.ctx.closed:
		putFrame(f)
		return errClosed
	}
}

func (c *Client) receiveFrame(frameType byte, reqID uuid.UUID) (*frame, error) {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID || reqID == uuid.Nil {
				if f.Type == errorFrame {
					putFrame(f)
					return nil, errFrame
				}
				if f.Type != frameType {
					putFrame(f)
					return nil, errInvalidFrame
				}
				return f, nil
			}
			putFrame(f)
		case <-c.ctx.closed:
			return nil, errClosed
		}
	}
}





func (c *Client) Send(r io.Reader) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	reqID := uuid.New()
	timeoutMs := uint64(defaultTimeout.Milliseconds())

	if err := c.sendFrame(startFrame, reqID, timeoutMs, flagRequest, nil, 0, false); err != nil {
		return err
	}
	if err := c.waitForAck(reqID); err != nil {
		return err
	}

	buf := getBuffer()
	defer putBuffer(buf)
	index := 0
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			isFinal := err == io.EOF
			if err := c.sendFrame(chunkFrame, reqID, timeoutMs, flagRequest, (*buf)[:n], index, isFinal); err != nil {
				return err
			}
			if err := c.waitForAck(reqID); err != nil {
				return err
			}
			index++
			if isFinal {
				break
			}
		}
		if err == io.EOF {
			if n == 0 {
				if err := c.sendFrame(chunkFrame, reqID, timeoutMs, flagRequest, nil, index, true); err != nil {
					return err
				}
				if err := c.waitForAck(reqID); err != nil {
					return err
				}
			}
			break
		}
		if err != nil {
			return err
		}
	}

	if err := c.sendFrame(endFrame, reqID, timeoutMs, flagRequest, nil, 0, false); err != nil {
		return err
	}
	return c.waitForAck(reqID)
}

func (c *Client) waitForAck(reqID uuid.UUID) error {
	for {
		select {
		case f := <-c.ctx.reads:
			if f.RequestID == reqID {
				if f.Type == ackFrame && f.Flags&flagAck != 0 {
					putFrame(f)
					return nil
				}
				if f.Type == errorFrame {
					errMsg := string(f.Payload)
					putFrame(f)
					return errors.New(errMsg)
				}
			}
			putFrame(f)
		case <-c.ctx.closed:
			return errClosed
		}
	}
}

func (c *Client) Receive(incoming chan io.Reader) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	rcvF, err := c.receiveFrame(startFrame, uuid.Nil)
	if err != nil {
		return err
	}
	reqID := rcvF.RequestID
	if err := c.sendAck(rcvF.FrameID, reqID); err != nil {
		putFrame(rcvF)
		return err
	}
	putFrame(rcvF)

	reqBuf, err := c.receiveAssembledChunkFramesWithAck(reqID)
	if err != nil {
		return err
	}

	endF, err := c.receiveFrame(endFrame, reqID)
	if err != nil {
		return err
	}
	if err := c.sendAck(endF.FrameID, reqID); err != nil {
		putFrame(endF)
		return err
	}
	putFrame(endF)

	incoming <- bytes.NewReader(reqBuf.Bytes())
	return nil
}

func (c *Client) Respond(incoming chan io.Reader, outgoing chan io.Reader) error {
	if err := c.Receive(incoming); err != nil {
		return err
	}
	reader := <-outgoing
	if reader == nil {
		return nil
	}
	return c.Send(reader)
}

func (c *Client) sendAck(frameID, reqID uuid.UUID) error {
	f := getFrame()
	f.Version = version1
	f.Type = ackFrame
	f.Flags = flagAck
	f.RequestID = reqID
	f.ClientID = c.clientID
	f.FrameID = frameID
	select {
	case c.ctx.writes <- f:
		return nil
	case <-c.ctx.closed:
		putFrame(f)
		return errClosed
	}
}

func (c *Client) receiveAssembledChunkFramesWithAck(reqID uuid.UUID) (bytes.Buffer, error) {
	var buff bytes.Buffer
	for {
		f, err := c.receiveFrame(chunkFrame, reqID)
		if err != nil {
			return buff, err
		}
		if err := c.sendAck(f.FrameID, reqID); err != nil {
			putFrame(f)
			return buff, err
		}
		buff.Write(f.Payload)
		isFinal := f.isFinal()
		putFrame(f)
		if isFinal {
			break
		}
	}
	return buff, nil
}

func (c *Client) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	select {
	case <-c.done:
		return nil
	default:
		close(c.done)
	}

	if c.ctx != nil {
		close(c.ctx.closed)
		if c.ctx.conn != nil {
			c.ctx.conn.Close()
		}
		c.ctx.wg.Wait()
		close(c.ctx.writes)
		close(c.ctx.errors)
		c.ctx.conn = nil
	}
	return nil
}
