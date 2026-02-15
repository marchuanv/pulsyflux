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
		addr:       addr,
		clientID:   uuid.New(),
		channelID:  channelID,
		done:       make(chan struct{}),
		requests:   make(chan *frame, 1024),
		responses:  make(chan *frame, 1024),
		registered: make(chan struct{}),
		session:    &session{incoming: make(chan *assembledRequest, 1024)},
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
	c.lastActive = time.Now()
	go c.idleMonitor()

	// Send registration frame to trigger server-side registration
	if err := c.sendRegistration(); err != nil {
		c.close()
		return nil, err
	}
	
	// Wait for registration to complete
	select {
	case <-c.registered:
	case <-time.After(2 * time.Second):
		c.close()
		return nil, errors.New("registration timeout")
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
func (c *Client) Stream(in io.Reader, out io.Writer, onComplete func(error)) {
	if onComplete == nil {
		onComplete = func(error) {}
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
		var err error
		if in != nil && out != nil {
			select {
			case req := <-c.session.incoming:
				out.Write(req.payload)
				err = c.doSend(in)
			case <-c.ctx.closed:
				err = errClosed
			}
		} else if in != nil {
			err = c.doSend(in)
		} else if out != nil {
			select {
			case req := <-c.session.incoming:
				out.Write(req.payload)
			case <-c.ctx.closed:
				err = errClosed
			}
		}
		onComplete(err)
	}()
}

func (c *Client) updateActivity() {
	c.activityMu.Lock()
	c.lastActive = time.Now()
	c.activityMu.Unlock()
}

func (c *Client) idleMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.activityMu.Lock()
			idle := time.Since(c.lastActive)
			c.activityMu.Unlock()
			if idle > 15*time.Second {
				c.close()
				return
			}
		case <-c.done:
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
		close(c.requests)
		close(c.responses)
		c.ctx.conn = nil
	}
	return nil
}
