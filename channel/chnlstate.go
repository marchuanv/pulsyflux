package channel

import (
	"errors"

	"github.com/google/uuid"
)

type ChannelState uuid.UUID

func newState(name string) ChannelState {
	state := ChannelState{}.create(name)
	return state
}

func (chSt ChannelState) create(name string) ChannelState {
	switch name {
	case "Open":
		id, _ := uuid.Parse("e273723e-e3a3-4766-b5d0-116f1b7d455a")
		return ChannelState(id)
	case "Closed":
		id, _ := uuid.Parse("ae7b49f8-4e67-4e1e-a2d0-5966dc3e914c")
		return ChannelState(id)
	case "Error":
		id, _ := uuid.Parse("96629304-0915-4d92-a944-79dc78b65359")
		return ChannelState(id)
	default:
		panic(errors.New("invalid channel state"))
	}
}

func (chSt ChannelState) name() string {
	switch chSt.string() {
	case "96629304-0915-4d92-a944-79dc78b65359":
		return "Error"
	case "ae7b49f8-4e67-4e1e-a2d0-5966dc3e914c":
		return "Closed"
	case "e273723e-e3a3-4766-b5d0-116f1b7d455a":
		return "Open"
	default:
		panic(errors.New("invalid channel state"))
	}
}

func (chSt ChannelState) string() string {
	return uuid.UUID(chSt).String()
}
