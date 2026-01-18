package socket

import "fmt"

type streamProcessor struct {
	RequestID uint64
	Total     int
}

func (p *streamProcessor) ProcessChunk(chunk []byte) error {
	p.Total += len(chunk)
	return nil
}

func (p *streamProcessor) Finish() (*frame, error) {
	msg := fmt.Sprintf("Processed %d bytes", p.Total)
	return &frame{
		Version:   Version1,
		Type:      MsgResponse,
		RequestID: p.RequestID,
		Payload:   []byte(msg),
	}, nil
}
