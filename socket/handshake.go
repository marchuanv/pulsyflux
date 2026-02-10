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

	reqID := uuid.New()
	if err := h.sendHandshakeFrame(reqID, uint64(defaultTimeout.Milliseconds()), flagHandshake); err != nil {
		return err
	}

	close(h.paired)
	return nil
}

func (h *handshakeClient) sendHandshakeFrame(reqID uuid.UUID, timeoutMs uint64, flags uint16) error {
	for retry := 0; retry < 5; retry++ {
		if retry > 0 {
			time.Sleep(200 * time.Millisecond)
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

		timeout := time.After(1 * time.Second)
		for {
			select {
			case resp := <-h.ctx.reads:
				if resp.Type == startFrame && resp.Flags == flagHandshake &&
					resp.ChannelID == h.channelID && resp.ClientID != h.clientID {
					h.peerID = resp.ClientID
					log.Printf("[Client %s] Handshake completed with peer %s", h.clientID, h.peerID)
					putFrame(resp)
					return nil
				}
				putFrame(resp)
			case <-timeout:
				goto nextRetry
			case <-h.done:
				return errClosed
			}
		}
		nextRetry:
	}
	return errPeerError
}
