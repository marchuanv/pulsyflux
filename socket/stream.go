package socket

import "fmt"

type streamprocessor struct {
	RequestID uint64
	Total     int
}

func (p *streamprocessor) ProcessChunk(chunk []byte) error {
	p.Total += len(chunk)
	// Add any per-chunk processing logic here
	return nil
}

func (p *streamprocessor) Finish() (*frame, error) {
	msg := fmt.Sprintf("Processed %d bytes", p.Total)
	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: p.RequestID,
		Payload:   []byte(msg),
	}, nil
}
