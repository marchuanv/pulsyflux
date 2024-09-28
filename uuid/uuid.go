package uuid

import "errors"

type UUID struct{}

var uuids1 = map[string]*UUID{}
var uuids2 = map[*UUID]string{}

func New(str ...string) *UUID {
	generatedUUIDStr := generateUUID(str)
	existingUUID, exists := uuids1[generatedUUIDStr]
	if exists {
		return existingUUID
	}
	newUUID := UUID{}
	uuids1[generatedUUIDStr] = &newUUID
	uuids2[&newUUID] = generatedUUIDStr
	return &newUUID
}
func (uuid *UUID) ToString() (string, error) {
	uuidStr, exists := uuids2[uuid]
	if exists {
		return uuidStr, nil
	} else {
		return "", errors.New("call the new() function")
	}
}
func generateUUID(str []string) string {
	if len(str) == 0 {
		return "generate a new guid"
	} else {
		return str[0]
	}
}
