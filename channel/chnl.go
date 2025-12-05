package channel

func (Id ChlId) HasChnl() bool {
	return chlReg.has(Id)
}

func (Id ChlId) HasSub(chlSubId ChlSubId) bool {
	return chlSubReg.has(Id, chlSubId)
}

func (Id ChlId) OpenChnl() {
	chlReg.register(Id)
}

func (Id ChlId) CloseChnl() {
	chlReg.unregister(Id)
}

func (Id ChlId) Publish(msg any) {
	nvlp := newChnlMsg(msg)
	chlSubReg.chlsubs(Id, func(sub *ChlSub) {
		go sub.rcvMsg(nvlp)
	})
}

func (Id ChlId) Subscribe(chlSubId ChlSubId, rcvMsg func(msg any)) {
	chlSubReg.register(Id, chlSubId)
	chlSubReg.get(Id, chlSubId, func(sub *ChlSub) {
		sub.rcvMsg = func(nvlp *chnlMsg) {
			canConv, cvrtMsg := getMsg[any](nvlp)
			if canConv {
				rcvMsg(cvrtMsg)
			}
		}
	})
}

func (Id ChlId) Unsubscribe(chlSubId ChlSubId) {
	chlSubReg.unregister(Id, chlSubId)
}
