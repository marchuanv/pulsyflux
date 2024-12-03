package task

import (
	"testing"
	"time"
)

func TestErrorHandle(test *testing.T) {
	failTest := true
	failTestMsg := ""
	ctx := NewTskCtx()
	ctx.Do(func(chl Channel) {
		chl.Write("Do_1")
		ctx.Do(func(chl Channel) {
			chl.Write("Do_1.1")
			ctx.Do(func(chl Channel) {
				chl.Write("Do_1.1.1")
				ctx.Do(func(chl Channel) {
					chl.Write("Do_1.1.1.1")
					panic("Do_1.1.1.1")
				}, func(err error, chl Channel) {
					chl.Write("Do_1.1.1.1_ErrorHandle")
				})
				ctx.Do(func(chl Channel) {
					chl.Write("Do_1.1.1.2")
					panic("Do_1.1.1.2")
				}, func(err error, chl Channel) {
					chl.Write("Do_1.1.1.2_ErrorHandle")
				})
			}, func(err error, chl Channel) {
				chl.Write("Do_1.1.1_ErrorHandle")
			})
			ctx.Do(func(chl Channel) {
				chl.Write("Do_1.2")
			})
		}, func(err error, chl Channel) {
			chl.Write("Do_1.1_ErrorHandle")
			ctx.Do(func(chl Channel) {
				chl.Write("Do_1.1_ErrorHandle_Do")
			})
		})
	}, func(err error, chl Channel) {
		chl.Write("Do_1_ErrorHandle")
		ctx.Do(func(chl Channel) {
			chl.Write("Do_1_ErrorHandle_Do")
		})
	})
	time.Sleep(5 * time.Second)
	if failTest {
		test.Log(failTestMsg)
		test.Fail()
	}
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
	ctx.Do(func(chl Channel) {
		ctx.Do(func(chl Channel) {
			ctx.Do(func(chl Channel) {
				ctx.Do(func(chl Channel) {
					panic("something went wrong")
				})
			})
		})
	})
}
