package socket

import (
	"log"
	"time"

	"github.com/google/uuid"
)

type handshakeClient struct {
	clientID  uuid.UUID
	peerID    uuid.UUID
	channelID uuid.UUID
	ctx       *connctx
	done      chan struct{}
	paired    chan struct{}
}

func (h *handshakeClient) doHandshake() error {
	select {
	case <-h.paired:
		return nil
	default:
	}

	if err := h.sendHandshakeFrame(uuid.New(), uint64(defaultTimeout.Milliseconds()), flagHandshake); err != nil {
		return err
	}

	close(h.paired)
	return nil
}

func (h *handshakeClient) sendHandshakeFrame(reqID uuid.UUID, timeoutMs uint64, flags uint16) error {
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			time.Sleep(time.Duration(retry) * time.Second)
		}

		f := getFrame()
		f.Version = version1
		f.Type = startFrame
		f.RequestID = reqID
		f.ClientID = h.clientID
		f.PeerClientID = h.peerID
		f.ChannelID = h.channelID
		f.ClientTimeoutMs = timeoutMs
		f.Flags = flags

		select {
		case h.ctx.writes <- f:
		case <-h.done:
			putFrame(f)
			return errClosed
		}

		for {
			select {
			case resp := <-h.ctx.reads:
				if resp.Type == errorFrame && resp.Flags&flagPeerNotAvailable != 0 && retry < 2 {
					putFrame(resp)
					break
				}

				if resp.Type == startFrame && resp.Flags == flagHandshake &&
					resp.ChannelID == h.channelID && resp.ClientID != h.clientID {
					h.peerID = resp.ClientID
					log.Printf("[Client %s] Handshake completed with peer %s", h.clientID, h.peerID)
					putFrame(resp)
					return nil
				}

				putFrame(resp)
				return errPeerError
			case <-h.done:
				return errClosed
			}
		}
	}
	return errPeerError
}
