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
	f.length = 1
	f.index = 0
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

func (c *Client) sendChunkFrame(reqID uuid.UUID, clientID uuid.UUID, timeoutMs uint64, index int, length int, payload []byte, flags uint16) error {
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
	f.index = index
	f.length = length
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
	f.index = 0
	f.length = 1
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

func (c *Client) receiveAssembledChunkFrames(reqId uuid.UUID, timeoutMs uint64) (payload io.Reader, requestId uuid.UUID, clientID uuid.UUID, err error) {
	var rcvF *frame
	var reqPayload []byte
	for {
		err = c.sendChunkFrame(reqId, c.clientID, timeoutMs, 0, 1, nil, flagReceive)
		if err != nil {
			return nil, requestId, clientID, err
		}
		rcvF, err = c.receiveChunkFrame(uuid.Nil)
		if err != nil {
			return nil, requestId, clientID, err
		}
		if rcvF.length == 0 || rcvF.length == (rcvF.index+1) { //last chunk frame
			break
		}
		requestId = rcvF.RequestID
		clientID = rcvF.ClientID
		reqPayload = append(reqPayload, rcvF.Payload...)
	}
	return bytes.NewReader(reqPayload), requestId, clientID, nil
}

func (c *Client) Send(r io.Reader, timeout time.Duration) (io.Reader, error) {

	var err error
	var rcvF *frame
	var n int

	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	err = c.sendStartFrame(reqID, c.clientID, timeoutMs, flagRequest)
	if err != nil {
		return nil, err
	}

	rcvF, err = c.receiveStartFrame(reqID)
	if err != nil {
		return nil, err
	}

	var respBuf bytes.Buffer
	buf := getBuffer()
	defer putBuffer(buf)

	totalFramesToExpect := 0 //need to determine how many chunk frames to expect based on payload size
	for {
		n, err = r.Read(*buf)
		if n > 0 {
			err = c.sendChunkFrame(reqID, c.clientID, timeoutMs, n, totalFramesToExpect, (*buf)[:n], flagRequest)
			if err != nil {
				return nil, err
			}
			rcvF, err = c.receiveChunkFrame(reqID)
			if err != nil {
				return nil, err
			}
			respBuf.Write(rcvF.Payload)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	err = c.sendEndFrame(reqID, c.clientID, timeoutMs, flagRequest)
	if err != nil {
		return nil, err
	}
	rcvF, err = c.receiveEndFrame(reqID)
	if err != nil {
		return nil, err
	}

	respBuf.Write(rcvF.Payload)

	return bytes.NewReader(respBuf.Bytes()), nil
}

func (c *Client) Receive(incoming chan io.Reader, outgoing chan io.Reader, timeout time.Duration) error {

	var n int
	var err error
	var rcvF *frame
	var reqID, reqClientID uuid.UUID

	origReqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	err = c.sendStartFrame(origReqID, c.clientID, timeoutMs, flagReceive)
	if err != nil {
		return err
	}

	rcvF, err = c.receiveStartFrame(uuid.Nil)
	if err != nil {
		return err
	}

	reqID = rcvF.RequestID
	reqClientID = rcvF.ClientID

	err = c.sendStartFrame(reqID, reqClientID, timeoutMs, flagResponse)
	if err != nil {
		return err
	}

	var reqPayload io.Reader
	reqPayload, reqID, reqClientID, err = c.receiveAssembledChunkFrames(origReqID, timeoutMs)
	if err != nil {
		return err
	}
	incoming <- reqPayload

	buf := getBuffer()
	defer putBuffer(buf)

	reader := <-outgoing
	totalFramesToExpect := 0 //need to determine how many chunk frames to expect based on payload size
	for {
		n, err = reader.Read(*buf)
		if n > 0 {
			payload := make([]byte, n)
			if n > 0 {
				copy(payload, (*buf)[:n])
			}
			err = c.sendChunkFrame(reqID, reqClientID, timeoutMs, n, totalFramesToExpect, payload, flagResponse)
			if err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
	}

	err = c.sendEndFrame(origReqID, c.clientID, timeoutMs, flagReceive)
	if err != nil {
		return err
	}

	rcvF, err = c.receiveEndFrame(uuid.Nil)
	if err != nil {
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
