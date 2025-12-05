package channel

import (
	"sync"

	"github.com/google/uuid"
)

type ChlId uuid.UUID
type ChlSubId uuid.UUID
type Chl struct {
	Id ChlId
	mu sync.Mutex
}
type ChlSub struct {
	Id     ChlSubId
	chlId  ChlId
	mu     sync.Mutex
	rcvMsg func(nvlp *chnlMsg)
}
type chlRegistry struct {
	chlSubReg *chlSubRegistry
}

type chlSubRegistry struct {
	chlReg *chlRegistry
}

var chlReg = &chlRegistry{}
var chlSubReg = &chlSubRegistry{}

func NewChl(uuidStr string) ChlId {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	if chlReg.chlSubReg == nil {
		chlReg.chlSubReg = chlSubReg
	}
	if chlSubReg.chlReg == nil {
		chlSubReg.chlReg = chlReg
	}
	return ChlId(Id)
}

func (chId ChlId) String() string {
	uuid := uuid.UUID(chId)
	return uuid.String()
}

func NewChlSub(uuidStr string) ChlSubId {
	Id, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}
	if chlReg.chlSubReg == nil {
		chlReg.chlSubReg = chlSubReg
	}
	if chlSubReg.chlReg == nil {
		chlSubReg.chlReg = chlReg
	}
	return ChlSubId(Id)
}

func (subId ChlSubId) String() string {
	uuid := uuid.UUID(subId)
	return uuid.String()
}
