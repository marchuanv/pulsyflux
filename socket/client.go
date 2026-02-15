package socket

import (
	"bytes"
	"errors"
	"io"
	"log"
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

	if err := c.register(); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) register() error {
	f := getFrame()
	f.Version = version1
	f.Type = startFrame
	f.Flags = flagReceive
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

func (c *Client) sendStartFrame(reqID uuid.UUID, clientID uuid.UUID, timeoutMs uint64, flags uint16) error {
	select {
	case <-c.ctx.closed:
		return errClosed
	default:
	}
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
	case <-c.ctx.closed:
		putFrame(f)
		return errClosed
	}
	return nil
}

func (c *Client) receiveStartFrame(reqID uuid.UUID) (*frame, error) {
	for {
		select {
		case f := <-c.ctx.reads:
			log.Printf("[Client %s] receiveStartFrame: got frame type=%d reqID=%s (expecting reqID=%s)", c.clientID.String()[:8], f.Type, f.RequestID.String()[:8], reqID.String()[:8])
			if f.RequestID == reqID || reqID == uuid.Nil {
				if f.Type == errorFrame {
					log.Printf("[Client %s] receiveStartFrame: ERROR FRAME", c.clientID.String()[:8])
					putFrame(f)
					return nil, errFrame
				}
				if f.Type != startFrame {
					log.Printf("[Client %s] receiveStartFrame: INVALID FRAME - expected startFrame(4), got type=%d", c.clientID.String()[:8], f.Type)
					putFrame(f)
					return nil, errInvalidFrame
				}
				log.Printf("[Client %s] receiveStartFrame: SUCCESS", c.clientID.String()[:8])
				return f, nil
			}
			log.Printf("[Client %s] receiveStartFrame: SKIPPING frame with wrong reqID", c.clientID.String()[:8])
			putFrame(f)
		case <-c.ctx.closed:
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
	case <-c.ctx.closed:
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
				return f, nil
			}
			putFrame(f)
		case <-c.ctx.closed:
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
	case <-c.ctx.closed:
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

	if err := c.sendStartFrame(reqID, c.clientID, timeoutMs, flagRequest); err != nil {
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
			if err := c.sendChunkFrame(reqID, c.clientID, timeoutMs, index, isFinal, (*buf)[:n], flagRequest); err != nil {
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
				if err := c.sendChunkFrame(reqID, c.clientID, timeoutMs, index, true, nil, flagRequest); err != nil {
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

	if err := c.sendEndFrame(reqID, c.clientID, timeoutMs, flagRequest); err != nil {
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

	log.Printf("[Client %s] Receive: Waiting for startFrame", c.clientID.String()[:8])
	rcvF, err := c.receiveStartFrame(uuid.Nil)
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

	endF, err := c.receiveEndFrame(reqID)
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
		f, err := c.receiveChunkFrame(reqID)
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
