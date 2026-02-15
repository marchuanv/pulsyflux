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
	incoming  chan *assembledRequest
	requests  chan *frame
	responses chan *frame
	opWg      sync.WaitGroup
	opErrors  chan error
	wg        sync.WaitGroup
	waitOnce  sync.Once
}

type assembledRequest struct {
	payload []byte
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
		incoming:  make(chan *assembledRequest, 16),
		requests:  make(chan *frame, 1024),
		responses: make(chan *frame, 1024),
		opErrors:  make(chan error, 100),
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
	c.wg.Add(2)
	go c.routeFrames()
	go c.processIncoming()

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





func (c *Client) Send(r io.Reader) {
	c.opWg.Add(1)
	go func() {
		defer c.opWg.Done()
		if err := c.doSend(r); err != nil {
			select {
			case c.opErrors <- err:
			default:
			}
		}
	}()
}

func (c *Client) doSend(r io.Reader) error {
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
		case f := <-c.responses:
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

func (c *Client) Receive(r io.Reader) {
	log.Printf("[Client %s] Receive: spawning goroutine", c.clientID.String()[:8])
	c.opWg.Add(1)
	go func() {
		defer c.opWg.Done()
		log.Printf("[Client %s] Receive: goroutine started", c.clientID.String()[:8])
		if err := c.doReceive(r); err != nil {
			select {
			case c.opErrors <- err:
			default:
			}
		}
	}()
}

func (c *Client) doReceive(r io.Reader) error {
	log.Printf("[Client %s] doReceive: waiting on c.incoming", c.clientID.String()[:8])
	select {
	case req := <-c.incoming:
		log.Printf("[Client %s] doReceive: received %d bytes from c.incoming", c.clientID.String()[:8], len(req.payload))
		if w, ok := r.(io.Writer); ok {
			w.Write(req.payload)
		}
		return nil
	case <-c.ctx.closed:
		return errClosed
	}
}

func (c *Client) Respond(req io.Reader, resp io.Reader) {
	c.opWg.Add(1)
	go func() {
		defer c.opWg.Done()
		if err := c.doRespond(req, resp); err != nil {
			select {
			case c.opErrors <- err:
			default:
			}
		}
	}()
}

func (c *Client) doRespond(req io.Reader, resp io.Reader) error {
	select {
	case r := <-c.incoming:
		if w, ok := req.(io.Writer); ok {
			w.Write(r.payload)
		}
		if resp != nil {
			c.Send(resp)
		}
		return nil
	case <-c.ctx.closed:
		return errClosed
	}
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

func (c *Client) routeFrames() {
	defer c.wg.Done()
	log.Printf("[Client %s] routeFrames: started", c.clientID.String()[:8])
	for {
		select {
		case f := <-c.ctx.reads:
			log.Printf("[Client %s] routeFrames: received frame type=%d flags=%d", c.clientID.String()[:8], f.Type, f.Flags)
			if f.Flags&flagRequest != 0 {
				select {
				case c.requests <- f:
					log.Printf("[Client %s] routeFrames: routed to requests", c.clientID.String()[:8])
				case <-c.ctx.closed:
					putFrame(f)
					return
				}
			} else {
				select {
				case c.responses <- f:
					log.Printf("[Client %s] routeFrames: routed to responses", c.clientID.String()[:8])
				case <-c.ctx.closed:
					putFrame(f)
					return
				}
			}
		case <-c.ctx.closed:
			return
		}
	}
}

func (c *Client) processIncoming() {
	defer c.wg.Done()
	log.Printf("[Client %s] processIncoming: started", c.clientID.String()[:8])
	for {
		select {
		case <-c.ctx.closed:
			return
		default:
		}

		log.Printf("[Client %s] processIncoming: waiting for startFrame", c.clientID.String()[:8])
		var startF *frame
		select {
		case startF = <-c.requests:
			log.Printf("[Client %s] processIncoming: received frame type=%d", c.clientID.String()[:8], startF.Type)
			if startF.Type != startFrame {
				log.Printf("[Client %s] processIncoming: not startFrame, discarding", c.clientID.String()[:8])
				putFrame(startF)
				continue
			}
		case <-c.ctx.closed:
			return
		}
		reqID := startF.RequestID
		startFrameID := startF.FrameID
		putFrame(startF)
		log.Printf("[Client %s] processIncoming: sending ack for startFrame", c.clientID.String()[:8])
		c.sendAck(startFrameID, reqID)

		var buf bytes.Buffer
		for {
			var chunkF *frame
			select {
			case chunkF = <-c.requests:
				if chunkF.Type != chunkFrame || chunkF.RequestID != reqID {
					putFrame(chunkF)
					continue
				}
			case <-c.ctx.closed:
				return
			}
			log.Printf("[Client %s] processIncoming: sending ack for chunkFrame", c.clientID.String()[:8])
			c.sendAck(chunkF.FrameID, reqID)
			buf.Write(chunkF.Payload)
			isFinal := chunkF.isFinal()
			putFrame(chunkF)
			if isFinal {
				break
			}
		}

		var endF *frame
		select {
		case endF = <-c.requests:
			if endF.Type != endFrame || endF.RequestID != reqID {
				putFrame(endF)
				continue
			}
		case <-c.ctx.closed:
			return
		}
		log.Printf("[Client %s] processIncoming: sending ack for endFrame", c.clientID.String()[:8])
		c.sendAck(endF.FrameID, reqID)
		putFrame(endF)

		log.Printf("[Client %s] processIncoming: queuing %d bytes to c.incoming", c.clientID.String()[:8], buf.Len())
		select {
		case c.incoming <- &assembledRequest{payload: buf.Bytes()}:
			log.Printf("[Client %s] processIncoming: queued successfully", c.clientID.String()[:8])
		case <-c.ctx.closed:
			return
		}
	}
}

func (c *Client) Wait() error {
	var firstErr error
	c.waitOnce.Do(func() {
		c.opWg.Wait()
		close(c.opErrors)
		for err := range c.opErrors {
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
	})
	return firstErr
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
		c.wg.Wait()
		close(c.ctx.writes)
		close(c.ctx.errors)
		close(c.incoming)
		close(c.requests)
		close(c.responses)
		c.ctx.conn = nil
	}
	return nil
}
