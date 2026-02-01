package socket

import (
	"bytes"
	"io"
	"sync"

	"github.com/google/uuid"
)

type providerRequest struct {
	ID     uuid.UUID
	Reader io.Reader
}

type providerResponse struct {
	ID     uuid.UUID
	Reader io.Reader
	Err    error
}

type provider struct {
	baseClient
	requests    chan *providerRequest
	responses   chan *providerResponse
	routingInfo map[uuid.UUID][]byte
	routingMu   sync.Mutex
	done        chan struct{}
}

func NewProvider(addr string, channelID uuid.UUID) (*provider, error) {
	p := &provider{
		baseClient: baseClient{
			addr:      addr,
			clientID:  uuid.New(),
			channelID: channelID,
			role:      RoleProvider,
		},
		requests:    make(chan *providerRequest, 100),
		responses:   make(chan *providerResponse, 100),
		routingInfo: make(map[uuid.UUID][]byte),
		done:        make(chan struct{}),
	}
	if err := p.dial(); err != nil {
		return nil, err
	}
	if err := p.register(); err != nil {
		p.close()
		return nil, err
	}
	go p.listen()
	go p.writeResponses()
	return p, nil
}

func (p *provider) listen() {
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

			// Store routing info for response
			p.routingMu.Lock()
			p.routingInfo[f.RequestID] = routing
			p.routingMu.Unlock()

			// Send request to channel for user to process
			req := &providerRequest{
				ID:     f.RequestID,
				Reader: bytes.NewReader(requestPayload),
			}

			select {
			case p.requests <- req:
			case <-p.done:
				return
			}
		}
	}
}

func (p *provider) writeResponses() {
	for {
		select {
		case <-p.done:
			return
		case resp := <-p.responses:
			p.routingMu.Lock()
			routing := p.routingInfo[resp.ID]
			delete(p.routingInfo, resp.ID)
			p.routingMu.Unlock()

			p.connMu.Lock()
			if p.conn == nil {
				p.connMu.Unlock()
				continue
			}

			if resp.Err != nil {
				errorPayload := append(routing, []byte(resp.Err.Error())...)
				errFrame := frame{
					Version:   Version1,
					Type:      ErrorFrame,
					RequestID: resp.ID,
					Payload:   errorPayload,
				}
				errFrame.write(p.conn)
				p.connMu.Unlock()
				continue
			}

			if err := p.sendChunkedResponse(resp.ID, resp.Reader, routing, ResponseStartFrame); err != nil {
				p.connMu.Unlock()
				continue
			}
			p.connMu.Unlock()
		}
	}
}

func (p *provider) Receive() (uuid.UUID, io.Reader, bool) {
	select {
	case req, ok := <-p.requests:
		if !ok {
			return uuid.Nil, nil, false
		}
		return req.ID, req.Reader, true
	case <-p.done:
		return uuid.Nil, nil, false
	}
}

func (p *provider) Respond(reqID uuid.UUID, r io.Reader, err error) {
	resp := &providerResponse{
		ID:     reqID,
		Reader: r,
		Err:    err,
	}
	select {
	case p.responses <- resp:
	case <-p.done:
	}
}

func (p *provider) Close() error {
	close(p.done)
	close(p.requests)
	close(p.responses)
	p.connMu.Lock()
	defer p.connMu.Unlock()
	return p.close()
}
