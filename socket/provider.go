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
	for {
		select {
		case <-p.done:
			return
		default:
		}

		f, err := newFrame(p.conn)
		if err != nil {
			return
		}

		if f.Type == StartFrame && f.Flags&FlagForwarded != 0 {
			// Extract routing info (consumer's clientID and channelID)
			if len(f.Payload) < 32 {
				continue
			}
			routingInfo := f.Payload[0:32]
			requestPayload := f.Payload[32:]

			response, err := p.handler(requestPayload)

			var respFrame frame
			if err != nil {
				// Prepend routing info to error message
				errorPayload := append(routingInfo, []byte(err.Error())...)
				respFrame = frame{
					Version:   Version1,
					Type:      ErrorFrame,
					RequestID: f.RequestID,
					Payload:   errorPayload,
				}
			} else {
				// Prepend routing info to response
				responsePayload := append(routingInfo, response...)
				respFrame = frame{
					Version:   Version1,
					Type:      ResponseFrame,
					RequestID: f.RequestID,
					Payload:   responsePayload,
				}
			}

			p.connMu.Lock()
			respFrame.write(p.conn)
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
