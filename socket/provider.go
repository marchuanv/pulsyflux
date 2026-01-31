package socket

import (
	"github.com/google/uuid"
)

type RequestHandler func(payload []byte) ([]byte, error)

type Provider struct {
	baseClient
	handler RequestHandler
	done    chan struct{}
}

func NewProvider(addr string, channelID uuid.UUID, handler RequestHandler) (*Provider, error) {
	p := &Provider{
		baseClient: baseClient{
			addr:      addr,
			clientID:  uuid.New(),
			channelID: channelID,
			role:      RoleProvider,
		},
		handler: handler,
		done:    make(chan struct{}),
	}
	if err := p.dial(); err != nil {
		return nil, err
	}
	if err := p.register(); err != nil {
		p.close()
		return nil, err
	}
	go p.listen()
	return p, nil
}

func (p *Provider) listen() {
	streamReqs := make(map[uuid.UUID][]byte)       // Stores request payload
	routingInfo := make(map[uuid.UUID][]byte)      // Stores routing info per request

	for {
		select {
		case <-p.done:
			return
		default:
		}

		p.connMu.Lock()
		if p.conn == nil {
			p.connMu.Unlock()
			return
		}
		conn := p.conn
		p.connMu.Unlock()

		f, err := newFrame(conn)
		if err != nil {
			return
		}

		switch f.Type {
		case StartFrame:
			if len(f.Payload) < 32 {
				panic("provider received invalid StartFrame with payload < 32 bytes")
			}
			// Extract and store routing info separately
			routingInfo[f.RequestID] = f.Payload[0:32]
			// Initialize empty payload for this request
			streamReqs[f.RequestID] = []byte{}

		case ChunkFrame:
			// Append chunk data to payload
			payload := streamReqs[f.RequestID]
			if payload == nil {
				// ChunkFrame arrived without StartFrame
				panic("provider received ChunkFrame without StartFrame")
			}
			streamReqs[f.RequestID] = append(payload, f.Payload...)

		case EndFrame:
			requestPayload := streamReqs[f.RequestID]
			routing := routingInfo[f.RequestID]
			delete(streamReqs, f.RequestID)
			delete(routingInfo, f.RequestID)

			response, err := p.handler(requestPayload)

			var respFrame frame
			if err != nil {
				errorPayload := append(routing, []byte(err.Error())...)
				respFrame = frame{
					Version:   Version1,
					Type:      ErrorFrame,
					RequestID: f.RequestID,
					Payload:   errorPayload,
				}
			} else {
				responsePayload := append(routing, response...)
				respFrame = frame{
					Version:   Version1,
					Type:      ResponseFrame,
					RequestID: f.RequestID,
					Payload:   responsePayload,
				}
			}

			p.connMu.Lock()
			conn := p.conn
			if conn != nil {
				if err := respFrame.write(conn); err != nil {
					// Failed to send response
				}
			}
			p.connMu.Unlock()
		}
	}
}

func (p *Provider) Close() error {
	close(p.done)
	p.connMu.Lock()
	defer p.connMu.Unlock()
	return p.close()
}
