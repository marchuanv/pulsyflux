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
	role      clientRole
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
	regFrame.Version = version1
	regFrame.Type = startFrame
	regFrame.Flags = flagRegistration
	regFrame.RequestID = reqID
	regFrame.Payload = c.buildMetadataPayload(uint64(5000))
	if err := regFrame.write(c.conn); err != nil {
		putFrame(regFrame)
		return err
	}
	putFrame(regFrame)

	_endFrame := getFrame()
	_endFrame.Version = version1
	_endFrame.Type = endFrame
	_endFrame.Flags = 0
	_endFrame.RequestID = reqID
	_endFrame.Payload = nil
	err := _endFrame.write(c.conn)
	putFrame(_endFrame)
	return err
}

func (c *baseClient) sendChunkedRequest(reqID uuid.UUID, r io.Reader) error {
	buf := getBuffer()
	defer putBuffer(buf)

	chunk := getFrame()
	chunk.Version = version1
	chunk.Type = chunkFrame
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
	_startFrame := getFrame()
	_startFrame.Version = version1
	_startFrame.Type = frameType
	_startFrame.Flags = 0
	_startFrame.RequestID = reqID
	_startFrame.Payload = routing
	if err := _startFrame.write(c.conn); err != nil {
		putFrame(_startFrame)
		return err
	}
	putFrame(_startFrame)

	buf := getBuffer()
	defer putBuffer(buf)

	chunk := getFrame()
	chunk.Version = version1
	chunk.Type = responseChunkFrame
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

	_endFrame := getFrame()
	_endFrame.Version = version1
	_endFrame.Type = responseEndFrame
	_endFrame.Flags = 0
	_endFrame.RequestID = reqID
	_endFrame.Payload = nil
	err := _endFrame.write(c.conn)
	putFrame(_endFrame)
	return err
}

func (c *baseClient) receiveChunkedResponse(reqID uuid.UUID) (io.Reader, error) {
	resp, err := newFrame(c.conn)
	if err != nil {
		return nil, err
	}
	if resp.RequestID != reqID {
		return nil, errRequestIDMismatch
	}
	if resp.Type == errorFrame {
		errorMsg := resp.Payload
		if len(errorMsg) >= 32 {
			errorMsg = errorMsg[32:]
		}
		return nil, errors.New(string(errorMsg))
	}
	if resp.Type != responseStartFrame {
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
			return nil, errRequestIDMismatch
		}
		switch f.Type {
		case responseChunkFrame:
			payload = append(payload, f.Payload...)
		case responseEndFrame:
			result := make([]byte, len(payload))
			copy(result, payload)
			putBuffer(buf)
			return bytes.NewReader(result), nil
		case errorFrame:
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
