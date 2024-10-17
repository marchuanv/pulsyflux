package util

import (
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
