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
	addr      string
	ctx       *connctx
	connMu    sync.Mutex
	clientID  uuid.UUID
	peerID    uuid.UUID
	channelID uuid.UUID
	role      clientRole
	done      chan struct{}
	handshake chan error
}

func NewClient(addr string, channelID uuid.UUID, role clientRole) (*Client, error) {
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
		role:      role,
		done:      make(chan struct{}),
		handshake: make(chan error, 1),
		ctx: &connctx{
			conn:   conn,
			writes: make(chan *frame, 1024),
			reads:  make(chan *frame, 100),
			errors: make(chan *frame, 256),
			closed: make(chan struct{}),
			wg:     &sync.WaitGroup{},
		},
	}

	c.ctx.wg.Add(3)
	go c.ctx.startWriter()
	go c.ctx.startReader()

	go func() { //Handshake in background
		defer c.ctx.wg.Done()
		handshakeReqID := uuid.New()
		timeoutMs := uint64(defaultTimeout.Milliseconds())
		err := c.sendStartFrame(handshakeReqID, timeoutMs, 0)
		if err != nil {
			log.Printf("[Client %s] Handshake failed: %v", c.clientID, err)
			c.handshake <- errHandshakeFailed
		} else {
			log.Printf("[Client %s] Handshake successful with peer %s", c.clientID, c.peerID)
			c.handshake <- nil
		}
	}()

	return c, nil
}

func (c *Client) sendStartFrame(reqID uuid.UUID, timeoutMs uint64, frameSeq int) error {
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			duration := time.Duration(retry) * time.Second
			time.Sleep(duration)
		}

		startF := getFrame()
		startF.Version = version1
		startF.Type = startFrame
		startF.RequestID = reqID
		startF.ClientID = c.clientID
		startF.PeerClientID = c.peerID
		startF.ChannelID = c.channelID
		startF.Role = c.role
		startF.ClientTimeoutMs = timeoutMs
		if c.peerID == uuid.Nil {
			startF.Flags = flagRegistration
		}
		log.Printf("[Client %s] Sending START frame [seq=%d] for request %s (retry=%d)", c.clientID, frameSeq, reqID, retry)
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
					if f.Flags&flagPeerNotAvailable != 0 && retry < 2 {
						log.Printf("[Client %s] No peers available, retrying...", c.clientID)
						putFrame(f)
						goto retryLoop
					}
					putFrame(f)
					return errPeerError
				case startFrame:
					if f.Flags == flagRegistration && c.peerID == uuid.Nil {
						log.Printf("[Client %s] Received start frame response for handshake from peer client %s", c.clientID, f.ClientID)
						if f.ChannelID == c.channelID && f.Role != c.role {
							c.peerID = f.ClientID
							log.Printf("[Client %s] Set peer ID to %s", c.clientID, c.peerID)
							putFrame(f)
							return nil
						} else if f.Role == c.role {
							log.Printf("[Client %s] ERROR: Role mismatch, peer has same role", c.clientID)
							putFrame(f)
							return errPeerError
						}
						putFrame(f)
						return nil
					} else if f.RequestID == reqID {
						log.Printf("[Client %s] Received start frame response for request %s", c.clientID, reqID)
						if c.peerID != f.ClientID {
							log.Printf("[Client %s] ERROR: Peer ID mismatch, expected %s, got %s", c.clientID, c.peerID, f.ClientID)
							putFrame(f)
							return errPeerError
						}
						putFrame(f)
						return nil
					} else {
						log.Printf("[Client %s] ERROR: Received unexpected start frame with request ID %s", c.clientID, f.RequestID)
						putFrame(f)
						return errPeerError
					}
				}
			case <-c.done:
				return errClosed
			}
		}
	retryLoop:
	}
	return errPeerError
}

func (c *Client) sendChunkFrame(reqID uuid.UUID, payload []byte, frameSeq int) (io.Reader, error) {
	chunk := getFrame()
	chunk.Version = version1
	chunk.Type = chunkFrame
	chunk.RequestID = reqID
	chunk.ClientID = c.clientID
	chunk.PeerClientID = c.peerID
	chunk.ChannelID = c.channelID
	chunk.Role = c.role
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
	endF.Role = c.role
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
	select {
	case err := <-c.handshake:
		if err != nil {
			return nil, err
		}
	case <-c.done:
		return nil, errClosed
	}

	reqID := uuid.New()
	timeoutMs := uint64(timeout.Milliseconds())
	if timeoutMs == 0 {
		timeoutMs = uint64(defaultTimeout.Milliseconds())
	}

	frameSeq := 0

	if err := c.sendStartFrame(reqID, timeoutMs, frameSeq); err != nil {
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

func (c *Client) Receive() (uuid.UUID, io.Reader, bool) {
	select {
	case err := <-c.handshake:
		if err != nil {
			return uuid.Nil, nil, false
		}
	case <-c.done:
		return uuid.Nil, nil, false
	}

	streamReqs := make(map[uuid.UUID]*bytes.Buffer)

	for {
		select {
		case f, ok := <-c.ctx.reads:
			if !ok {
				return uuid.Nil, nil, false
			}

			switch f.Type {
			case startFrame:
				streamReqs[f.RequestID] = bytes.NewBuffer(make([]byte, 0, 1024))
				putFrame(f)

			case chunkFrame:
				buf := streamReqs[f.RequestID]
				if buf != nil {
					buf.Write(f.Payload)
				}
				putFrame(f)

			case endFrame:
				buf := streamReqs[f.RequestID]
				delete(streamReqs, f.RequestID)
				reqID := f.RequestID
				putFrame(f)
				if buf != nil {
					return reqID, bytes.NewReader(buf.Bytes()), true
				}

			case errorFrame:
				log.Printf("[Client %s] ERROR: Received error frame for request %s: %s", c.clientID, f.RequestID, string(f.Payload))
				delete(streamReqs, f.RequestID)
				putFrame(f)
				return uuid.Nil, nil, false
			}

		case <-c.done:
			return uuid.Nil, nil, false
		}
	}
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
