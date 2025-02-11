package channel

import (
	"github.com/google/uuid"
)

type chnlEnvlp func() (any, uuid.UUID)

func newChnlEnvlp(content any) *chnlEnvlp {
	Id := uuid.New()
	_chMsgF := chnlEnvlp(func() (any, uuid.UUID) {
		return content, Id
	})
	return &_chMsgF
}
func (chEnv chnlEnvlp) content() (any, uuid.UUID) {
	return chEnv()
}

func getEnvlpContent[T any](envlp *chnlEnvlp) (canConvert bool, content T) {
	defer (func() {
		err := recover()
		if err != nil {
			canConvert = false
		}
	})()

	cont, _ := envlp.content()
	content = cont.(T)
	canConvert = true
	return canConvert, content
}
