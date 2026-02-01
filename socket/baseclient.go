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

	// Use a reasonable timeout for waiting for peers during registration
	regFrame := frame{
		Version:   Version1,
		Type:      StartFrame,
		Flags:     FlagRegistration,
		RequestID: reqID,
		Payload:   c.buildMetadataPayload(uint64(5000)),
	}
	if err := regFrame.write(c.conn); err != nil {
		return err
	}

	endFrame := frame{
		Version:   Version1,
		Type:      EndFrame,
		Flags:     0,
		RequestID: reqID,
	}
	return endFrame.write(c.conn)
}

func (c *baseClient) sendChunkedRequest(reqID uuid.UUID, r io.Reader) error {
	buf := make([]byte, maxFrameSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := frame{
				Version:   Version1,
				Type:      ChunkFrame,
				Flags:     0,
				RequestID: reqID,
				Payload:   buf[:n],
			}
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
	startFrame := frame{
		Version:   Version1,
		Type:      frameType,
		Flags:     0,
		RequestID: reqID,
		Payload:   routing,
	}
	if err := startFrame.write(c.conn); err != nil {
		return err
	}

	buf := make([]byte, maxFrameSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := frame{
				Version:   Version1,
				Type:      ResponseChunkFrame,
				Flags:     0,
				RequestID: reqID,
				Payload:   buf[:n],
			}
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

	endFrame := frame{
		Version:   Version1,
		Type:      ResponseEndFrame,
		Flags:     0,
		RequestID: reqID,
	}
	return endFrame.write(c.conn)
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

	var payload []byte
	for {
		f, err := newFrame(c.conn)
		if err != nil {
			return nil, err
		}
		if f.RequestID != reqID {
			return nil, ErrRequestIDMismatch
		}
		switch f.Type {
		case ResponseChunkFrame:
			payload = append(payload, f.Payload...)
		case ResponseEndFrame:
			return bytes.NewReader(payload), nil
		case ErrorFrame:
			errorMsg := f.Payload
			if len(errorMsg) >= 32 {
				errorMsg = errorMsg[32:]
			}
			return nil, errors.New(string(errorMsg))
		default:
			return nil, errors.New("unexpected response frame type in chunk stream")
		}
	}
}
