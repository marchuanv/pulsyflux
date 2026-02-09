package socket

import (
	"log"
	"time"

	"github.com/google/uuid"
)

type handshakeClient struct {
	clientID   uuid.UUID
	peerID     uuid.UUID
	channelID  uuid.UUID
	ctx        *connctx
	done       chan struct{}
	handshake  chan struct{}
	paired     chan struct{}
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
				if f.Type == errorFrame {
					log.Printf("[Client %s] ERROR: Received error frame during handshake", h.clientID)
					if f.Flags&flagPeerNotAvailable != 0 && retry < 2 {
						log.Printf("[Client %s] No peers available, retrying...", h.clientID)
						putFrame(f)
						goto retryLoop
					}
					putFrame(f)
					return errPeerError
				}
				if f.Type == startFrame && f.Flags == flags {
					if flags == flagHandshakeStarted && h.peerID == uuid.Nil {
						if f.ChannelID == h.channelID && f.ClientID != h.clientID {
							h.peerID = f.ClientID
							log.Printf("[Client %s] Set peer ID to %s", h.clientID, h.peerID)
						}
					}
					if f.RequestID == reqID || f.Flags == flags {
						log.Printf("[Client %s] Received handshake response", h.clientID)
						putFrame(f)
						return nil
					}
				}
				putFrame(f)
			case <-h.done:
				return errClosed
			}
		}
	retryLoop:
	}
	return errPeerError
}
