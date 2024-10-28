package msgbus

import "encoding/gob"

func BootStrap() {
	m := msg{}
	gob.Register(m)
	gob.Register(Msg(m))
}
