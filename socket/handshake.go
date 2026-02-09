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

	// Send handshake started
	handshakeStartedReqID := uuid.New()
	if err := h.sendHandshakeFrame(handshakeStartedReqID, timeoutMs, flagHandshakeStarted); err != nil {
		return err
	}

	// Send handshake completed
	handshakeCompletedReqID := uuid.New()
	if err := h.sendHandshakeFrame(handshakeCompletedReqID, timeoutMs, flagHandshakeCompleted); err != nil {
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
					log.Printf("[Client %s] ERROR: Received error frame during handshake", h.clientID)
					if f.Flags&flagPeerNotAvailable != 0 && retry < 2 {
						log.Printf("[Client %s] No peers available, retrying...", h.clientID)
						putFrame(f)
						goto retryLoop
					}
					putFrame(f)
					return errPeerError
				case startFrame:
					switch f.Flags {
					case flagHandshakeStarted:
						if f.ChannelID == h.channelID && f.ClientID != h.clientID {
							h.peerID = f.ClientID
							log.Printf("[Client %s] Set peer ID to %s", h.clientID, h.peerID)
							putFrame(f)
							return nil
						}
						log.Printf("[Client %s] ERROR: Received handshake startFrame with invalid channel ID or client ID", h.clientID)
						putFrame(f)
						return errPeerError
					case flagHandshakeCompleted:
						if h.peerID == uuid.Nil {
							log.Printf("[Client %s] ERROR: Handshake completed but no peer ID set", h.clientID)
							return errPeerError
						}
						log.Printf("[Client %s] Handshake completed successfully with peer %s", h.clientID, h.peerID)
						putFrame(f)
						return nil
					default:
						log.Printf("[Client %s] ERROR: Received unexpected frame type %d during handshake", h.clientID, f.Type)
						putFrame(f)
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
