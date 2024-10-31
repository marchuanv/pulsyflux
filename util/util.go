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

func GetHostAndPortFromAddress(address string) *Address {
	return Do(func() (*Address, error) {
		hostStr, portStr, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			return nil, err
		}
		address := &Address{hostStr, port}
		return address, err
	})
}

func IsValidUUID(u string) bool {
	return Do(func() (bool, error) {
		_, err := uuid.Parse(u)
		isValid := (err == nil)
		return isValid, err
	})
}

func IsEmptyString(str string) bool {
	return str == ""
}
func StringFromReader(reader io.ReadCloser) string {
	return Do(func() (string, error) {
		output, err := io.ReadAll(reader)
		outputStr := string(output)
		if err != nil {
			return outputStr, err
		}
		err = reader.Close()
		if err != nil {
			return outputStr, err
		}
		return outputStr, err
	})
}

func ReaderFromString(str string) (io.Reader, error) {
	reader := bytes.NewReader([]byte(str))
	return reader, nil
}

func Deserialise[T any](serialised string) T {
	return Do(func() (T, error) {
		var decoded T
		if len(serialised) == 0 {
			return decoded, errors.New("the serialised argument is an empty string")
		}
		by, err := base64.StdEncoding.DecodeString(serialised)
		if err != nil {
			return decoded, err
		}
		buf := bytes.NewBuffer(by)
		d := gob.NewDecoder(buf)
		err = d.Decode(&decoded)
		return decoded, err
	})
}

func Serialise[T any](e T) string {
	return Do(func() (string, error) {
		var base64Str string
		buf := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)
		err := enc.Encode(&e)
		if err != nil {
			return base64Str, err
		}
		base64Str = base64.StdEncoding.EncodeToString(buf.Bytes())
		return base64Str, nil
	})
}
