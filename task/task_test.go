package task

import (
	"fmt"
	"pulsyflux/channel"
	"testing"
)

func TestErrorHandle(test *testing.T) {
	ctx := NewTskCtx()
	chnl := ctx.Do(func(chl channel.Channel) {
		chl.Write("Do_1")
		ctx.Do(func(chl channel.Channel) {
			chl.Write("Do_1.1")
			ctx.Do(func(chl channel.Channel) {
				chl.Write("Do_1.1.1")
				ctx.Do(func(chl channel.Channel) {
					chl.Write("Do_1.1.1.1")
					panic("Do_1.1.1.1")
				}, func(err error, chl channel.Channel) {
					chl.Write("Do_1.1.1.1_ErrorHandle")
				})
				ctx.Do(func(chl channel.Channel) {
					chl.Write("Do_1.1.1.2")
					panic("Do_1.1.1.2")
				}, func(err error, chl channel.Channel) {
					chl.Write("Do_1.1.1.2_ErrorHandle")
				})
			}, func(err error, chl channel.Channel) {
				chl.Write("Do_1.1.1_ErrorHandle")
			})
			ctx.Do(func(chl channel.Channel) {
				chl.Write("Do_1.2")
			})
		}, func(err error, chl channel.Channel) {
			chl.Write("Do_1.1_ErrorHandle")
			ctx.Do(func(chl channel.Channel) {
				chl.Write("Do_1.1_ErrorHandle_Do")
			})
		})
	}, func(err error, chl channel.Channel) {
		chl.Write("Do_1_ErrorHandle")
		ctx.Do(func(chl channel.Channel) {
			chl.Write("Do_1_ErrorHandle_Do")
		})
	})
	chnl.Read(func(data any) {
		if data != nil {
			if data == "Do_1_ErrorHandle_Do" {
				fmt.Printf("\r\n%s\r\n", data)
			}
		}
	})
}

func TestPanicNoErrorHandle(test *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			test.Errorf("The implementation did not panic")
			test.Fail()
		}
	}()
	ctx := NewTskCtx()
	ctx.Do(func(chl channel.Channel) {
		ctx.Do(func(chl channel.Channel) {
			ctx.Do(func(chl channel.Channel) {
				ctx.Do(func(chl channel.Channel) {
					panic("something went wrong")
				})
			})
		})
	})
}
