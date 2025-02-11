package util

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"io"
	"pulsyflux/task"

	"github.com/google/uuid"
)

func Newv5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}

func IsValidUUID(u string) bool {
	return task.DoNow(u, func(id string) bool {
		_, err := uuid.Parse(id)
		isValid := (err == nil)
		return isValid
	})
}

func IsEmptyString(str string) bool {
	return str == ""
}

func StringFromReader(reader io.ReadCloser) string {
	return task.DoNow(reader, func(r io.ReadCloser) string {
		output, err := io.ReadAll(r)
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
	return task.DoNow(str, func(s string) io.Reader {
		return bytes.NewReader([]byte(s))
	})
}

func Deserialise[T any](serialised string) T {
	return task.DoNow(serialised, func(str string) T {
		var decoded T
		if len(str) == 0 {
			panic(errors.New("the serialised argument is an empty string"))
		}
		by, err := base64.StdEncoding.DecodeString(str)
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
	return task.DoNow(e, func(t T) string {
		gob.Register(t)
		buf := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)
		err := enc.Encode(&t)
		if err != nil {
			panic(err)
		}
		return base64.StdEncoding.EncodeToString(buf.Bytes())
	})
}
