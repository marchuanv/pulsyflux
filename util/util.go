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

type Address struct {
	host string
	port int
}

func Newv5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}

func GetHostAndPortFromAddress(address string) *Result[Address] {
	return Do(true, func() (*Result[Address], error) {
		hostStr, portStr, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			return nil, err
		}
		address := Address{hostStr, port}
		result := &Result[Address]{address}
		return result, err
	})
}
func IsValidUUID(u string) *Result[bool] {
	return Do(true, func() (*Result[bool], error) {
		_, err := uuid.Parse(u)
		isValid := (err == nil)
		result := &Result[bool]{isValid}
		return result, err
	})
}
func IsEmptyString(str string) bool {
	return str == ""
}
func StringFromReader(reader io.ReadCloser) *Result[string] {
	return Do(true, func() (*Result[string], error) {
		output, err := io.ReadAll(reader)
		result := &Result[string]{string(output)}
		if err != nil {
			return result, err
		}
		err = reader.Close()
		if err != nil {
			return result, err
		}
		return result, err
	})
}

func ReaderFromString(str string) (io.Reader, error) {
	reader := bytes.NewReader([]byte(str))
	return reader, nil
}

func Deserialise[T any](serialised string) *Result[T] {
	return Do(true, func() (*Result[T], error) {
		if len(serialised) == 0 {
			return nil, errors.New("the serialised argument is an empty string")
		}
		by, err := base64.StdEncoding.DecodeString(serialised)
		if err != nil {
			return nil, err
		}
		buf := bytes.NewBuffer(by)
		d := gob.NewDecoder(buf)
		var decoded T
		err = d.Decode(&decoded)
		result := &Result[T]{decoded}
		return result, err
	})
}

func Serialise[T any](e T) *Result[string] {
	return Do(true, func() (*Result[string], error) {
		buf := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)
		err := enc.Encode(&e)
		if err != nil {
			return nil, err
		}
		base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
		result := &Result[string]{base64Str}
		return result, nil
	})
}
