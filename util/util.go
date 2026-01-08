package util

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"io"

	"github.com/google/uuid"
)

func Newv5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}

func IsValidUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	isValid := (err == nil)
	return isValid
}

func IsEmptyString(str string) bool {
	return str == ""
}

func StringFromReader(reader io.ReadCloser) (string, error) {
	output, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	outputStr := string(output)
	err = reader.Close()
	if err != nil {
		panic(err)
	}
	return outputStr, nil
}

func ReaderFromString(str string) io.Reader {
	return bytes.NewReader([]byte(str))
}

func Deserialise[T any](serialised string) T {
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
}

func Serialise[T any](e T) string {
	gob.Register(e)
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(&e)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
