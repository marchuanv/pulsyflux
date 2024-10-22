package util

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/google/uuid"
)

func Newv5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}

func GetHostAndPortFromAddress(address string) (string, int, error) {
	hostStr, portStr, hostPortErr := net.SplitHostPort(address)
	if hostPortErr != nil {
		return "", 0, hostPortErr
	}
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil {
		return "", 0, convErr
	}
	return hostStr, port, nil
}
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
func IsEmptyString(str string) bool {
	return str == ""
}
func StringFromReader(reader io.ReadCloser) (string, error) {
	output, err1 := io.ReadAll(reader)
	if err1 != nil {
		return "", err1
	}
	err2 := reader.Close()
	if err2 != nil {
		return "", err2
	}
	return string(output), nil
}

func ReaderFromString(str string) (io.Reader, error) {
	reader := bytes.NewReader([]byte(str))
	return reader, nil
}

func Deserialise(serialised string) (any, error) {
	if len(serialised) == 0 {
		return nil, errors.New("the serialised argument is an empty string")
	}
	by, err := base64.StdEncoding.DecodeString(serialised)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(by)
	d := gob.NewDecoder(buf)
	var decoded any
	err = d.Decode(&decoded)
	return decoded, err
}

func Serialise(e any) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(&e)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
