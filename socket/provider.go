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

type Provider struct {
	baseClient
	requests    chan *providerRequest
	responses   chan *providerResponse
	routingInfo map[uuid.UUID][]byte
	routingMu   sync.Mutex
	done        chan struct{}
}

func NewProvider(addr string, channelID uuid.UUID) (*Provider, error) {
	p := &Provider{
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

func (p *Provider) listen() {
	streamReqs := make(map[uuid.UUID]*bytes.Buffer)  // Use buffer for efficient appending
	routingInfo := make(map[uuid.UUID][]byte)        // Stores routing info per request

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
				putFrame(f)
				continue // Skip invalid frames instead of panic
			}
			routingInfo[f.RequestID] = f.Payload[0:32]
			streamReqs[f.RequestID] = bytes.NewBuffer(make([]byte, 0, 1024))
			putFrame(f)

		case ChunkFrame:
			buf := streamReqs[f.RequestID]
			if buf == nil {
				putFrame(f)
				continue // Skip instead of panic
			}
			buf.Write(f.Payload)
			putFrame(f)

		case EndFrame:
			buf := streamReqs[f.RequestID]
			routing := routingInfo[f.RequestID]
			delete(streamReqs, f.RequestID)
			delete(routingInfo, f.RequestID)
			putFrame(f)

			if buf == nil {
				continue
			}

			p.routingMu.Lock()
			p.routingInfo[f.RequestID] = routing
			p.routingMu.Unlock()

			req := &providerRequest{
				ID:     f.RequestID,
				Reader: bytes.NewReader(buf.Bytes()),
			}

			select {
			case p.requests <- req:
			case <-p.done:
				return
			}
		}
	}
}

func (p *Provider) writeResponses() {
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
				errFrame := getFrame()
				errFrame.Version = Version1
				errFrame.Type = ErrorFrame
				errFrame.RequestID = resp.ID
				errFrame.Payload = errorPayload
				errFrame.write(p.conn)
				putFrame(errFrame)
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

func (p *Provider) Receive() (uuid.UUID, io.Reader, bool) {
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

func (p *Provider) Respond(reqID uuid.UUID, r io.Reader, err error) {
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

func (p *Provider) Close() error {
	close(p.done)
	close(p.requests)
	close(p.responses)
	p.connMu.Lock()
	defer p.connMu.Unlock()
	return p.close()
}
