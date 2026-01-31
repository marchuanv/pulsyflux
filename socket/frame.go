package socket

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
)

const Version1 byte = 1

const (
	ResponseFrame            byte = 0x02
	ErrorFrame               byte = 0x03
	StartFrame               byte = 0x04
	ChunkFrame               byte = 0x05
	EndFrame                 byte = 0x06
	frameHeaderSize               = 24
	maxFrameSize                  = 1024 * 1024
	defaultFrameReadTimeout       = 2 * time.Minute
	defaultFrameWriteTimeout      = 5 * time.Second
)

// Frame flags
const (
	FlagForwarded uint16 = 1 << 0 // Indicates this frame was forwarded by server
)

type frame struct {
	Version   byte
	Type      byte
	Flags     uint16
	RequestID uuid.UUID // 16-byte UUID
	Payload   []byte
}

// newFrame reads a frame from a net.Conn
func newFrame(conn net.Conn) (*frame, error) {
	conn.SetReadDeadline(time.Now().Add(defaultFrameReadTimeout))

	var header [frameHeaderSize]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, err
	}

	if header[0] != Version1 {
		return nil, errors.New("unsupported protocol version")
	}

	payloadLen := binary.BigEndian.Uint32(header[20:24])
	if payloadLen > maxFrameSize {
		return nil, errors.New("frame payload too large")
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	f := &frame{
		Version: header[0],
		Type:    header[1],
		Flags:   binary.BigEndian.Uint16(header[2:4]),
		Payload: payload,
	}
	copy(f.RequestID[:], header[4:20])

	return f, nil
}

// newErrorFrame constructs an error frame with the given message
func newErrorFrame(reqID uuid.UUID, msg string) *frame {
	return &frame{
		Version:   Version1,
		Type:      ErrorFrame,
		RequestID: reqID,
		Payload:   []byte(msg),
	}
}

// write writes the frame to a net.Conn
func (f *frame) write(conn net.Conn) error {
	conn.SetWriteDeadline(time.Now().Add(defaultFrameWriteTimeout))

	var header [frameHeaderSize]byte
	header[0] = f.Version
	header[1] = f.Type
	binary.BigEndian.PutUint16(header[2:4], f.Flags)
	copy(header[4:20], f.RequestID[:])
	binary.BigEndian.PutUint32(header[20:24], uint32(len(f.Payload)))

	if _, err := conn.Write(header[:]); err != nil {
		return err
	}

	_, err := conn.Write(f.Payload)
	return err
}
