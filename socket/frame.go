package socket

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type frame struct {
	Version   byte
	Type      byte
	Flags     uint16
	RequestID uint64
	Payload   []byte
}

// readFrame reads a single frame from conn
func readFrame(conn net.Conn) (*frame, error) {
	var header [headerSize]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, err
	}

	if header[0] != Version1 {
		return nil, errors.New("unsupported protocol version")
	}

	length := binary.BigEndian.Uint32(header[12:16])
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	return &frame{
		Version:   header[0],
		Type:      header[1],
		Flags:     binary.BigEndian.Uint16(header[2:4]),
		RequestID: binary.BigEndian.Uint64(header[4:12]),
		Payload:   payload,
	}, nil
}

// writeFrame writes a single frame to conn
func writeFrame(conn net.Conn, f *frame) error {
	if f.Type == MsgRequest && len(f.Payload) > maxFrameSize {
		return errors.New("inline payload too large; use streaming")
	}

	var header [headerSize]byte
	header[0] = f.Version
	header[1] = f.Type
	binary.BigEndian.PutUint16(header[2:4], f.Flags)
	binary.BigEndian.PutUint64(header[4:12], f.RequestID)
	binary.BigEndian.PutUint32(header[12:16], uint32(len(f.Payload)))

	if _, err := conn.Write(header[:]); err != nil {
		return err
	}
	_, err := conn.Write(f.Payload)
	return err
}

func errorFrame(reqID uint64, msg string) *frame {
	return &frame{
		Version:   Version1,
		Type:      MsgError,
		RequestID: reqID,
		Payload:   []byte(msg),
	}
}
