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
	addr       string
	clientID   uuid.UUID
	channelID  uuid.UUID
	ctx        *connctx
	done       chan struct{}
	connMu     sync.Mutex
	sessionMu  sync.Mutex
	requests   chan *frame
	responses  chan *frame
	wg         sync.WaitGroup
	registered chan struct{}
	session    *session
	lastActive time.Time
	activityMu sync.Mutex
}

type session struct {
	incoming chan *assembledRequest
}

type assembledRequest struct {
	payload []byte
}

func NewClient(addr string, channelID uuid.UUID) (*Client, error) {
	c := &Client{
		addr:      addr,
		channelID: channelID,
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) connect() error {
	conn, err := net.Dial("tcp4", c.addr)
	if err != nil {
		return err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetReadBuffer(2 * 1024 * 1024)
		tcpConn.SetWriteBuffer(2 * 1024 * 1024)
	}

	c.clientID = uuid.New()
	c.done = make(chan struct{})
	c.requests = make(chan *frame, 1024)
	c.responses = make(chan *frame, 1024)
	c.registered = make(chan struct{})
	c.session = &session{incoming: make(chan *assembledRequest, 1024)}
	c.ctx = &connctx{
		conn:   conn,
		writes: make(chan *frame, 1024),
		reads:  make(chan *frame, 1024),
		errors: make(chan *frame, 256),
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}

	c.ctx.wg.Add(2)
	go c.ctx.startWriter()
	go c.ctx.startReader()
	c.wg.Add(2)
	go c.routeFrames()
	go c.processIncoming()
	c.lastActive = time.Now()
	go c.idleMonitor()

	if err := c.sendRegistration(); err != nil {
		c.close()
		return err
	}
	
	select {
	case <-c.registered:
		return nil
	case <-time.After(2 * time.Second):
		c.close()
		return errors.New("registration timeout")
	}
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
func (c *Client) BroadcastStream(in io.Reader, onComplete func(error)) {
	if onComplete == nil {
		onComplete = func(error) {}
	}
	
	if err := c.ensureConnected(); err != nil {
		onComplete(err)
		return
	}
	
	c.updateActivity()
	c.sessionMu.Lock()
	select {
	case <-c.session.incoming:
		c.sessionMu.Unlock()
		onComplete(errors.New("session has unconsumed messages"))
		return
	default:
	}
	c.session = &session{incoming: make(chan *assembledRequest, 1024)}
	c.sessionMu.Unlock()

	go func() {
		onComplete(c.doSend(in))
	}()
}

func (c *Client) SubscriptionStream(out io.Writer, onComplete func(error)) {
	if onComplete == nil {
		onComplete = func(error) {}
	}
	
	if err := c.ensureConnected(); err != nil {
		onComplete(err)
		return
	}
	
	c.updateActivity()
	c.sessionMu.Lock()
	select {
	case <-c.session.incoming:
		c.sessionMu.Unlock()
		onComplete(errors.New("session has unconsumed messages"))
		return
	default:
	}
	c.session = &session{incoming: make(chan *assembledRequest, 1024)}
	sess := c.session
	ctxClosed := c.ctx.closed
	c.sessionMu.Unlock()

	go func() {
		var err error
		select {
		case req := <-sess.incoming:
			out.Write(req.payload)
		case <-ctxClosed:
			err = errClosed
		}
		onComplete(err)
	}()
}

func (c *Client) updateActivity() {
	c.activityMu.Lock()
	c.lastActive = time.Now()
	c.activityMu.Unlock()
}

func (c *Client) ensureConnected() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	
	select {
	case <-c.done:
		if c.ctx != nil {
			c.ctx.wg.Wait()
			c.wg.Wait()
		}
		return c.reconnect()
	default:
		return nil
	}
}

func (c *Client) reconnect() error {
	conn, err := net.Dial("tcp4", c.addr)
	if err != nil {
		return err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetReadBuffer(2 * 1024 * 1024)
		tcpConn.SetWriteBuffer(2 * 1024 * 1024)
	}

	c.clientID = uuid.New()
	c.done = make(chan struct{})
	c.requests = make(chan *frame, 1024)
	c.responses = make(chan *frame, 1024)
	c.registered = make(chan struct{})
	c.session = &session{incoming: make(chan *assembledRequest, 1024)}
	c.wg = sync.WaitGroup{}
	c.ctx = &connctx{
		conn:   conn,
		writes: make(chan *frame, 1024),
		reads:  make(chan *frame, 1024),
		errors: make(chan *frame, 256),
		closed: make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}

	c.ctx.wg.Add(2)
	go c.ctx.startWriter()
	go c.ctx.startReader()
	c.wg.Add(2)
	go c.routeFrames()
	go c.processIncoming()
	c.lastActive = time.Now()
	go c.idleMonitor()

	if err := c.sendRegistration(); err != nil {
		c.closeWithoutLock()
		return err
	}
	
	select {
	case <-c.registered:
		return nil
	case <-time.After(2 * time.Second):
		c.closeWithoutLock()
		return errors.New("registration timeout")
	}
}

func (c *Client) idleMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.activityMu.Lock()
			idle := time.Since(c.lastActive)
			c.activityMu.Unlock()
			if idle > 5*time.Second {
				c.close()
				return
			}
		case <-c.ctx.closed:
			return
		}
	}
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


func (c *Client) routeFrames() {
	defer c.wg.Done()
	for {
		select {
		case f := <-c.ctx.reads:
			// Check for registration ack
			if f.Type == ackFrame && f.Flags == flagNone {
				select {
				case c.registered <- struct{}{}:
				default:
				}
				putFrame(f)
				continue
			}
			if f.Flags&flagRequest != 0 {
				select {
				case c.requests <- f:
				case <-c.ctx.closed:
					putFrame(f)
					return
				}
			} else {
				select {
				case c.responses <- f:
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
	for {
		select {
		case <-c.ctx.closed:
			return
		default:
		}

		var startF *frame
		select {
		case startF = <-c.requests:
			if startF.Type != startFrame {
				putFrame(startF)
				continue
			}
		case <-c.ctx.closed:
			return
		}
		reqID := startF.RequestID
		startFrameID := startF.FrameID
		putFrame(startF)
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
		c.sendAck(endF.FrameID, reqID)
		putFrame(endF)

		select {
		case c.session.incoming <- &assembledRequest{payload: buf.Bytes()}:
		case <-c.ctx.closed:
			return
		}
	}
}



func (c *Client) close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.closeWithoutLock()
}

func (c *Client) closeWithoutLock() error {
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
	}
	return nil
}
