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
	handshake chan struct{}
	paired    chan struct{}
}

func (h *handshakeClient) doHandshake() error {
	select {
	case <-h.paired:
		return nil
	default:
	}

	timeoutMs := uint64(defaultTimeout.Milliseconds())

	// Single handshake frame
	handshakeReqID := uuid.New()
	if err := h.sendHandshakeFrame(handshakeReqID, timeoutMs, flagHandshake); err != nil {
		return err
	}

	close(h.handshake)
	close(h.paired)
	return nil
}

func (h *handshakeClient) sendHandshakeFrame(reqID uuid.UUID, timeoutMs uint64, flags uint16) error {
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			duration := time.Duration(retry) * time.Second
			time.Sleep(duration)
		}

		startF := getFrame()
		startF.Version = version1
		startF.Type = startFrame
		startF.RequestID = reqID
		startF.ClientID = h.clientID
		startF.PeerClientID = h.peerID
		startF.ChannelID = h.channelID
		startF.ClientTimeoutMs = timeoutMs
		startF.Flags = flags
		log.Printf("[Client %s] Sending handshake frame for request %s (retry=%d)", h.clientID, reqID, retry)
		select {
		case h.ctx.writes <- startF:
		case <-h.done:
			putFrame(startF)
			return errClosed
		}
		for {
			select {
			case f := <-h.ctx.reads:
				switch f.Type {
				case errorFrame:
					if f.Flags&flagPeerNotAvailable != 0 && retry < 2 {
						log.Printf("[Client %s] No peers available, retrying...", h.clientID)
						putFrame(f)
						goto retryLoop
					}
					log.Printf("[Client %s] ERROR: Received error frame during handshake", h.clientID)
					putFrame(f)
					return errPeerError
				case startFrame:
					switch f.Flags {
					case flagHandshake:
						if f.ChannelID == h.channelID && f.ClientID != h.clientID {
							h.peerID = f.ClientID
							log.Printf("[Client %s] Handshake completed with peer %s", h.clientID, h.peerID)
							putFrame(f)
							return nil
						}
						log.Printf("[Client %s] ERROR: Received handshake frame with invalid channel ID or client ID", h.clientID)
						putFrame(f)
						return errPeerError
					default:
						log.Printf("[Client %s] ERROR: Received unexpected handshake frame with flags %d", h.clientID, f.Flags)
						return errPeerError
					}
				default:
					log.Printf("[Client %s] ERROR: Received unexpected frame type %d during handshake", h.clientID, f.Type)
					putFrame(f)
					return errPeerError
				}
			case <-h.done:
				return errClosed
			}
		}
	retryLoop:
	}
	return errPeerError
}
