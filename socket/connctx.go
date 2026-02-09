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

type clientRole byte

const (
	roleConsumer clientRole = 0x01
	roleProvider clientRole = 0x02
)

const version1 byte = 1

const (
	errorFrame               byte   = 0x03
	startFrame               byte   = 0x04
	chunkFrame               byte   = 0x05
	endFrame                 byte   = 0x06
	frameHeaderSize                 = 81
	maxFrameSize                    = 1024 * 1024
	defaultFrameReadTimeout         = 2 * time.Minute
	defaultFrameWriteTimeout        = 5 * time.Second
	flagRegistration         uint16 = 0x01
	flagPeerNotAvailable     uint16 = 0x02
	defaultClientTimeoutMs   uint64 = 30000 // 30 seconds default timeout
)

type connctx struct {
	conn   net.Conn
	writes chan *frame
	reads  chan *frame
	errors chan *frame // Higher priority for error frames
	closed chan struct{}
	wg     *sync.WaitGroup
}

type frame struct {
	Version         byte
	Type            byte
	Flags           uint16
	RequestID       uuid.UUID // 16-byte UUID
	ClientID        uuid.UUID // 16-byte UUID
	PeerClientID    uuid.UUID // 16-byte UUID for peer routing
	ChannelID       uuid.UUID // 16-byte UUID for channel
	ClientTimeoutMs uint64    // Added to frame for timeout tracking
	Role            clientRole
	Payload         []byte
}

// newErrorFrame constructs an error frame with the given message and flags
func newErrorFrame(reqID uuid.UUID, clientID uuid.UUID, msg string, flags uint16) *frame {
	f := getFrame()
	f.Version = version1
	f.Type = errorFrame
	f.Flags = flags
	f.RequestID = reqID
	f.ClientID = clientID
	f.PeerClientID = uuid.Nil
	f.ClientTimeoutMs = defaultClientTimeoutMs
	f.Payload = []byte(msg)
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

	payloadLen := binary.BigEndian.Uint32(header[77:81])
	if payloadLen > maxFrameSize {
		return nil, errors.New("frame payload too large")
	}

	f := getFrame()
	f.Version = header[0]
	f.Type = header[1]
	f.Flags = binary.BigEndian.Uint16(header[2:4])
	copy(f.RequestID[:], header[4:20])
	copy(f.ClientID[:], header[20:36])
	copy(f.PeerClientID[:], header[36:52])
	copy(f.ChannelID[:], header[52:68])
	f.ClientTimeoutMs = binary.BigEndian.Uint64(header[68:76])
	f.Role = clientRole(header[76])

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
	copy(header[36:52], f.PeerClientID[:])
	copy(header[52:68], f.ChannelID[:])
	binary.BigEndian.PutUint64(header[68:76], f.ClientTimeoutMs)
	header[76] = byte(f.Role)
	binary.BigEndian.PutUint32(header[77:81], uint32(len(f.Payload)))
	// Single write for header+payload reduces syscalls
	if len(f.Payload) > 0 {
		// Use buffer pool for small payloads
		if len(f.Payload) <= 8192 {
			buf := getWriteBuffer()
			defer putWriteBuffer(buf)
			copy((*buf)[:frameHeaderSize], header[:])
			copy((*buf)[frameHeaderSize:], f.Payload)
			_, err := ctx.conn.Write((*buf)[:frameHeaderSize+len(f.Payload)])
			return err
		}
		// Large payloads: allocate once
		buf := make([]byte, frameHeaderSize+len(f.Payload))
		copy(buf, header[:])
		copy(buf[frameHeaderSize:], f.Payload)
		_, err := ctx.conn.Write(buf)
		return err
	}
	_, err := ctx.conn.Write(header[:])
	return err
}
