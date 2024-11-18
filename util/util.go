package util

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"io"
	"net"
	"pulsyflux/task"
	"strconv"

	"github.com/google/uuid"
)

type Address struct {
	Host string
	Port int
}

func Newv5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}

func NewAddress(address string) *Address {
	return task.DoNow[*Address, any](func() *Address {
		hostStr, portStr, err := net.SplitHostPort(address)
		if err != nil {
			panic(err)
		}
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			panic(convErr)
		}
		address := &Address{hostStr, port}
		return address
	})
}

func (add *Address) String() string {
	return add.Host + ":" + strconv.Itoa(add.Port)
}

func IsValidUUID(u string) bool {
	return task.DoNow[bool, any](func() bool {
		_, err := uuid.Parse(u)
		isValid := (err == nil)
		return isValid
	})
}

func IsEmptyString(str string) bool {
	return str == ""
}

func StringFromReader(reader io.ReadCloser) string {
	return task.DoNow[string, any](func() string {
		output, err := io.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		outputStr := string(output)
		err = reader.Close()
		if err != nil {
			panic(err)
		}
		return outputStr
	}, nil)
}

func ReaderFromString(str string) io.Reader {
	return task.DoNow[io.Reader, any](func() io.Reader {
		return bytes.NewReader([]byte(str))
	})
}

func Deserialise[T any](serialised string) T {
	return task.DoNow[T, any](func() T {
		var decoded T
		if len(serialised) == 0 {
			panic(errors.New("the serialised argument is an empty string"))
		}
		by, err := base64.StdEncoding.DecodeString(serialised)
		if err != nil {
			panic(err)
		}
		buf := bytes.NewBuffer(by)
		d := gob.NewDecoder(buf)
		err = d.Decode(&decoded)
		if err != nil {
			panic(err)
		}
		return decoded
	})
}

func Serialise[T any](e T) string {
	return task.DoNow[string, any](func() string {
		gob.Register(e)
		var base64Str string
		buf := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)
		err := enc.Encode(&e)
		if err != nil {
			panic(err)
		}
		base64Str = base64.StdEncoding.EncodeToString(buf.Bytes())
		return base64Str
	})
}
