package socket

import (
	"encoding/binary"
	"encoding/json"
	"errors"
)

type requestmeta struct {
	TimeoutMs uint32 `json:"timeout_ms,omitempty"`
	DataSize  uint64 `json:"data_size"`
	Type      string `json:"type,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	Streaming bool   `json:"streaming,omitempty"`
}

func (meta *requestmeta) payload(data []byte) ([]byte, error) {
	if meta.DataSize == 0 {
		meta.DataSize = uint64(len(data))
	}

	// Skip size check if data is nil (used for meta-only frames)
	if data != nil && uint64(len(data)) != meta.DataSize {
		return nil, errors.New("data_size mismatch")
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	if len(metaBytes) > 64*1024 {
		return nil, errors.New("meta too large")
	}

	buf := make([]byte, 4+len(metaBytes)+len(data))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(metaBytes))) // meta length
	copy(buf[4:], metaBytes)
	if data != nil {
		copy(buf[4+len(metaBytes):], data)
	}

	return buf, nil
}
