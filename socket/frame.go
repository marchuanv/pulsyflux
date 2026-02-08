package socket

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
)

const version1 byte = 1

const (
	responseFrame            byte   = 0x02
	errorFrame               byte   = 0x03
	startFrame               byte   = 0x04
	chunkFrame               byte   = 0x05
	endFrame                 byte   = 0x06
	responseStartFrame       byte   = 0x07
	responseChunkFrame       byte   = 0x08
	responseEndFrame         byte   = 0x09
	frameHeaderSize                 = 40
	maxFrameSize                    = 1024 * 1024
	defaultFrameReadTimeout         = 2 * time.Minute
	defaultFrameWriteTimeout        = 5 * time.Second
	flagRegistration         uint16 = 0x01
)

type frame struct {
	Version   byte
	Type      byte
	Flags     uint16
	RequestID uuid.UUID // 16-byte UUID
	ClientID  uuid.UUID // 16-byte UUID
	Payload   []byte
}

// newFrame reads a frame from a net.Conn
func newFrame(conn net.Conn) (*frame, error) {
	conn.SetReadDeadline(time.Now().Add(defaultFrameReadTimeout))

	var header [frameHeaderSize]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, err
	}

	if header[0] != version1 {
		return nil, errors.New("unsupported protocol version")
	}

	payloadLen := binary.BigEndian.Uint32(header[36:40])
	if payloadLen > maxFrameSize {
		return nil, errors.New("frame payload too large")
	}

	f := getFrame()
	f.Version = header[0]
	f.Type = header[1]
	f.Flags = binary.BigEndian.Uint16(header[2:4])
	copy(f.RequestID[:], header[4:20])
	copy(f.ClientID[:], header[20:36])

	if payloadLen > 0 {
		f.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(conn, f.Payload); err != nil {
			putFrame(f)
			return nil, err
		}
	} else {
		f.Payload = nil
	}

	return f, nil
}

// newErrorFrame constructs an error frame with the given message
func newErrorFrame(reqID uuid.UUID, clientID uuid.UUID, msg string) *frame {
	f := getFrame()
	f.Version = version1
	f.Type = errorFrame
	f.RequestID = reqID
	f.ClientID = clientID
	f.Payload = []byte(msg)
	return f
}

// write writes the frame to a net.Conn
func (f *frame) write(conn net.Conn) error {
	conn.SetWriteDeadline(time.Now().Add(defaultFrameWriteTimeout))

	var header [frameHeaderSize]byte
	header[0] = f.Version
	header[1] = f.Type
	binary.BigEndian.PutUint16(header[2:4], f.Flags)
	copy(header[4:20], f.RequestID[:])
	copy(header[20:36], f.ClientID[:])
	binary.BigEndian.PutUint32(header[36:40], uint32(len(f.Payload)))

	// Single write for header+payload reduces syscalls
	if len(f.Payload) > 0 {
		// Use buffer pool for small payloads
		if len(f.Payload) <= 8192 {
			buf := getWriteBuffer()
			defer putWriteBuffer(buf)
			copy((*buf)[:frameHeaderSize], header[:])
			copy((*buf)[frameHeaderSize:], f.Payload)
			_, err := conn.Write((*buf)[:frameHeaderSize+len(f.Payload)])
			return err
		}
		// Large payloads: allocate once
		buf := make([]byte, frameHeaderSize+len(f.Payload))
		copy(buf, header[:])
		copy(buf[frameHeaderSize:], f.Payload)
		_, err := conn.Write(buf)
		return err
	}

	_, err := conn.Write(header[:])
	return err
}
