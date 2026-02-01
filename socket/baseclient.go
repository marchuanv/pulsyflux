package socket

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/google/uuid"
)

type baseClient struct {
	addr      string
	conn      net.Conn
	connMu    sync.Mutex
	clientID  uuid.UUID
	channelID uuid.UUID
	role      ClientRole
}

func (c *baseClient) dial() error {
	conn, err := net.Dial("tcp4", c.addr)
	if err != nil {
		return err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetReadBuffer(2 * 1024 * 1024)
		tcpConn.SetWriteBuffer(2 * 1024 * 1024)
	}
	c.conn = conn
	return nil
}

func (c *baseClient) close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *baseClient) buildMetadataPayload(timeoutMs uint64) []byte {
	payload := make([]byte, 1+8+16+16)
	payload[0] = byte(c.role)
	binary.BigEndian.PutUint64(payload[1:9], timeoutMs)
	copy(payload[9:25], c.clientID[:])
	copy(payload[25:41], c.channelID[:])
	return payload
}

func (c *baseClient) register() error {
	reqID := uuid.New()

	regFrame := getFrame()
	regFrame.Version = Version1
	regFrame.Type = StartFrame
	regFrame.Flags = FlagRegistration
	regFrame.RequestID = reqID
	regFrame.Payload = c.buildMetadataPayload(uint64(5000))
	if err := regFrame.write(c.conn); err != nil {
		putFrame(regFrame)
		return err
	}
	putFrame(regFrame)

	endFrame := getFrame()
	endFrame.Version = Version1
	endFrame.Type = EndFrame
	endFrame.Flags = 0
	endFrame.RequestID = reqID
	endFrame.Payload = nil
	err := endFrame.write(c.conn)
	putFrame(endFrame)
	return err
}

func (c *baseClient) sendChunkedRequest(reqID uuid.UUID, r io.Reader) error {
	buf := getBuffer()
	defer putBuffer(buf)
	
	chunk := getFrame()
	chunk.Version = Version1
	chunk.Type = ChunkFrame
	chunk.Flags = 0
	chunk.RequestID = reqID
	defer putFrame(chunk)
	
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			chunk.Payload = (*buf)[:n]
			if err := chunk.write(c.conn); err != nil {
				return err
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

func (c *baseClient) sendChunkedResponse(reqID uuid.UUID, r io.Reader, routing []byte, frameType byte) error {
	startFrame := getFrame()
	startFrame.Version = Version1
	startFrame.Type = frameType
	startFrame.Flags = 0
	startFrame.RequestID = reqID
	startFrame.Payload = routing
	if err := startFrame.write(c.conn); err != nil {
		putFrame(startFrame)
		return err
	}
	putFrame(startFrame)

	buf := getBuffer()
	defer putBuffer(buf)
	
	chunk := getFrame()
	chunk.Version = Version1
	chunk.Type = ResponseChunkFrame
	chunk.Flags = 0
	chunk.RequestID = reqID
	defer putFrame(chunk)
	
	for {
		n, err := r.Read(*buf)
		if n > 0 {
			chunk.Payload = (*buf)[:n]
			if err := chunk.write(c.conn); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	endFrame := getFrame()
	endFrame.Version = Version1
	endFrame.Type = ResponseEndFrame
	endFrame.Flags = 0
	endFrame.RequestID = reqID
	endFrame.Payload = nil
	err := endFrame.write(c.conn)
	putFrame(endFrame)
	return err
}

func (c *baseClient) receiveChunkedResponse(reqID uuid.UUID) (io.Reader, error) {
	resp, err := newFrame(c.conn)
	if err != nil {
		return nil, err
	}
	if resp.RequestID != reqID {
		return nil, ErrRequestIDMismatch
	}
	if resp.Type == ErrorFrame {
		errorMsg := resp.Payload
		if len(errorMsg) >= 32 {
			errorMsg = errorMsg[32:]
		}
		return nil, errors.New(string(errorMsg))
	}
	if resp.Type != ResponseStartFrame {
		return nil, errors.New("unexpected frame type")
	}

	// Use buffer pool for response assembly
	buf := getBuffer()
	payload := (*buf)[:0]
	
	for {
		f, err := newFrame(c.conn)
		if err != nil {
			putBuffer(buf)
			return nil, err
		}
		if f.RequestID != reqID {
			putBuffer(buf)
			return nil, ErrRequestIDMismatch
		}
		switch f.Type {
		case ResponseChunkFrame:
			payload = append(payload, f.Payload...)
		case ResponseEndFrame:
			result := make([]byte, len(payload))
			copy(result, payload)
			putBuffer(buf)
			return bytes.NewReader(result), nil
		case ErrorFrame:
			putBuffer(buf)
			errorMsg := f.Payload
			if len(errorMsg) >= 32 {
				errorMsg = errorMsg[32:]
			}
			return nil, errors.New(string(errorMsg))
		default:
			putBuffer(buf)
			return nil, errors.New("unexpected response frame type in chunk stream")
		}
	}
}
