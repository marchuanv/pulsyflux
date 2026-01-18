package socket

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

type requestmeta struct {
	TimeoutMs uint32 `json:"timeout_ms,omitempty"`
	DataSize  uint64 `json:"data_size"`
	Type      string `json:"type,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	Streaming bool   `json:"streaming,omitempty"`
}

// --------------------- OLD method for backward compatibility (optional) ---------------------
func (meta *requestmeta) payload(data []byte) ([]byte, error) {
	if meta.DataSize == 0 {
		meta.DataSize = uint64(len(data))
	}

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

// --------------------- NEW package-private helper functions ---------------------

// encodeRequestMeta encodes a requestmeta into a length-prefixed byte slice
func encodeRequestMeta(meta requestmeta) ([]byte, error) {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	if len(metaBytes) > 64*1024 {
		return nil, errors.New("meta too large")
	}

	buf := make([]byte, 4+len(metaBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(metaBytes)))
	copy(buf[4:], metaBytes)

	return buf, nil
}

// decodeRequestMeta decodes a length-prefixed requestmeta from a reader
func decodeRequestMeta(r io.Reader) (*requestmeta, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
		return nil, err
	}

	metaLen := binary.BigEndian.Uint32(lengthBuf[:])
	metaBytes := make([]byte, metaLen)
	if _, err := io.ReadFull(r, metaBytes); err != nil {
		return nil, err
	}

	var meta requestmeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
