package socket

import (
	"encoding/binary"
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

	regFrame := frame{
		Version:   Version1,
		Type:      StartFrame,
		Flags:     0,
		RequestID: reqID,
		Payload:   c.buildMetadataPayload(0),
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
