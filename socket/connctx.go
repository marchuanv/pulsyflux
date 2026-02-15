package socket

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

const version1 byte = 1

const (
	errorFrame               byte   = 0x03
	startFrame               byte   = 0x04
	chunkFrame               byte   = 0x05
	endFrame                 byte   = 0x06
	ackFrame                 byte   = 0x07
	frameHeaderSize                 = 84
	maxFrameSize                    = 1024 * 1024
	defaultFrameReadTimeout         = 2 * time.Minute
	defaultFrameWriteTimeout        = 5 * time.Second
	flagNone                 uint16 = 0x00
	flagRequest              uint16 = 0x01
	flagAck                  uint16 = 0x02
	flagReceive              uint16 = 0x04
	flagResponse             uint16 = 0x08
	defaultClientTimeoutMs   uint64 = 30000
)

type connctx struct {
	conn   net.Conn
	writes chan *frame
	reads  chan *frame
	errors chan *frame
	closed chan struct{}
	wg     *sync.WaitGroup
}

type frame struct {
	Version         byte
	Type            byte
	Flags           uint16
	RequestID       uuid.UUID
	ClientID        uuid.UUID
	ChannelID       uuid.UUID
	FrameID         uuid.UUID
	ClientTimeoutMs uint64
	Payload         []byte
	sequence        uint32
}

const sequenceFinalFlag uint32 = 0x80000000

func (f *frame) setSequence(index int, isFinal bool) {
	f.sequence = uint32(index)
	if isFinal {
		f.sequence |= sequenceFinalFlag
	}
}

func (f *frame) getIndex() int {
	return int(f.sequence & ^sequenceFinalFlag)
}

func (f *frame) isFinal() bool {
	return (f.sequence & sequenceFinalFlag) != 0
}

func newErrorFrame(reqID uuid.UUID, clientID uuid.UUID, msg string, flags uint16) *frame {
	f := getFrame()
	f.Version = version1
	f.Type = errorFrame
	f.Flags = flags
	f.RequestID = reqID
	f.ClientID = clientID
	f.ClientTimeoutMs = defaultClientTimeoutMs
	f.Payload = []byte(msg)
	return f
}

func cloneFrame(src *frame) *frame {
	f := getFrame()
	f.Version = src.Version
	f.Type = src.Type
	f.Flags = src.Flags
	f.RequestID = src.RequestID
	f.ClientID = src.ClientID
	f.ChannelID = src.ChannelID
	f.FrameID = src.FrameID
	f.ClientTimeoutMs = src.ClientTimeoutMs
	f.sequence = src.sequence
	if len(src.Payload) > 0 {
		f.Payload = make([]byte, len(src.Payload))
		copy(f.Payload, src.Payload)
	}
	return f
}

func (ctx *connctx) enqueue(f *frame) bool {
	defer func() {
		// Recover from panic if channel is closed
		recover()
	}()

	// Send error frames on priority channel
	if f.Type == errorFrame {
		select {
		case ctx.errors <- f:
			return true
		case <-ctx.closed:
			return false
		default:
			return false
		}
	}

	// Send regular frames on normal channel
	select {
	case ctx.writes <- f:
		return true
	case <-ctx.closed:
		return false
	default:
		return false // drop frame if write buffer full
	}
}

func (ctx *connctx) startReader() {
	defer ctx.wg.Done()
	for {
		select {
		case <-ctx.closed:
			return
		default:
		}

		f, err := ctx.readFrame()
		if err != nil {
			return
		}

		select {
		case ctx.reads <- f:
		case <-ctx.closed:
			putFrame(f)
			return
		}
	}
}

func (ctx *connctx) startWriter() {
	defer ctx.wg.Done()
	for {
		select {
		// Prioritize error frames
		case frame, ok := <-ctx.errors:
			if !ok {
				// errors channel closed, drain writes and exit
				for {
					select {
					case frame, ok := <-ctx.writes:
						if !ok {
							return
						}
						_ = ctx.writeFrame(frame)
					case <-ctx.closed:
						return
					}
				}
			}
			if frame != nil {
				_ = ctx.writeFrame(frame)
			}
		case frame, ok := <-ctx.writes:
			if !ok {
				return
			}
			_ = ctx.writeFrame(frame)
		case <-ctx.closed:
			return
		}
	}
}

func (ctx *connctx) readFrame() (*frame, error) {
	ctx.conn.SetReadDeadline(time.Now().Add(defaultFrameReadTimeout))

	var header [frameHeaderSize]byte
	if _, err := io.ReadFull(ctx.conn, header[:]); err != nil {
		return nil, err
	}

	if header[0] != version1 {
		return nil, errors.New("unsupported protocol version")
	}

	payloadLen := binary.BigEndian.Uint32(header[80:84])
	if payloadLen > maxFrameSize {
		return nil, errors.New("frame payload too large")
	}

	f := getFrame()
	f.Version = header[0]
	f.Type = header[1]
	f.Flags = binary.BigEndian.Uint16(header[2:4])
	copy(f.RequestID[:], header[4:20])
	copy(f.ClientID[:], header[20:36])
	copy(f.ChannelID[:], header[36:52])
	copy(f.FrameID[:], header[52:68])
	f.ClientTimeoutMs = binary.BigEndian.Uint64(header[68:76])
	f.sequence = binary.BigEndian.Uint32(header[76:80])

	if payloadLen > 0 {
		f.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(ctx.conn, f.Payload); err != nil {
			putFrame(f)
			return nil, err
		}
	} else {
		f.Payload = nil
	}

	return f, nil
}

func (ctx *connctx) writeFrame(f *frame) error {
	ctx.conn.SetWriteDeadline(time.Now().Add(defaultFrameWriteTimeout))
	var header [frameHeaderSize]byte
	header[0] = f.Version
	header[1] = f.Type
	binary.BigEndian.PutUint16(header[2:4], f.Flags)
	copy(header[4:20], f.RequestID[:])
	copy(header[20:36], f.ClientID[:])
	copy(header[36:52], f.ChannelID[:])
	copy(header[52:68], f.FrameID[:])
	binary.BigEndian.PutUint64(header[68:76], f.ClientTimeoutMs)
	binary.BigEndian.PutUint32(header[76:80], f.sequence)
	binary.BigEndian.PutUint32(header[80:84], uint32(len(f.Payload)))
	if len(f.Payload) > 0 {
		if len(f.Payload) <= 8192 {
			buf := getWriteBuffer()
			defer putWriteBuffer(buf)
			copy((*buf)[:frameHeaderSize], header[:])
			copy((*buf)[frameHeaderSize:], f.Payload)
			_, err := ctx.conn.Write((*buf)[:frameHeaderSize+len(f.Payload)])
			return err
		}
		buf := make([]byte, frameHeaderSize+len(f.Payload))
		copy(buf, header[:])
		copy(buf[frameHeaderSize:], f.Payload)
		_, err := ctx.conn.Write(buf)
		return err
	}
	_, err := ctx.conn.Write(header[:])
	return err
}
