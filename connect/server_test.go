package connect

import (
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"testing"
	"time"
)

func TestHttpServer(test *testing.T) {

	var httpServStartedCh *msgbus.Channel
	var startHttpServCh *msgbus.Channel
	// var stopHttpServCh *msgbus.Channel
	// var failedToStartHttpServCh *msgbus.Channel
	// var failedToStopHttpServCh *msgbus.Channel
	// var httpServPortUnavCh *msgbus.Channel
	// var httpServerResCh *msgbus.Channel
	// var httpServerSuccResCh *msgbus.Channel
	// var httpServerErrResCh *msgbus.Channel

	HttpServerSubscriptions()
	httpCh := msgbus.New(subscriptions.HTTP)

	// httpServerResCh = httpCh.New(subscriptions.HTTP_SERVER_RESPONSE)
	// //HTTP RESPONSE SUCCESS
	// go (func() {
	// 	httpServerSuccResCh = httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
	// 	successHttpResponseMsg := httpServerSuccResCh.Subscribe()
	// 	test.Log(successHttpResponseMsg)
	// })()
	// //HTTP RESPONSE FAIL
	// go (func() {
	// 	httpServerErrResCh = httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)
	// 	failedHttpResponseMsg := httpServerErrResCh.Subscribe()
	// 	test.Log(failedHttpResponseMsg)
	// })()

	// //FAILED TO START EVENT
	// go (func() {
	// 	go (func() {
	// 		httpServPortUnavCh = startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
	// 		httpServPortUnavMsg := httpServPortUnavCh.Subscribe()
	// 		test.Log(httpServPortUnavMsg)
	// 	})()
	// 	failedToStartHttpServCh = startHttpServCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
	// 	failedToStartMsg := failedToStartHttpServCh.Subscribe()
	// 	test.Log(failedToStartMsg)
	// })()

	//STARTED EVENT
	go (func() {
		httpServStartedCh = httpCh.New(subscriptions.HTTP_SERVER_STARTED)
		startedMsg := httpServStartedCh.Subscribe()
		test.Log(startedMsg)
	})()

	// //STOPPED EVENT
	// go (func() {
	// 	go (func() {
	// 		failedToStopHttpServCh = stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
	// 		failedToStopHttpServMsg := failedToStopHttpServCh.Subscribe()
	// 		test.Log(failedToStopHttpServMsg)
	// 	})()
	// 	stopHttpServCh = httpCh.New(subscriptions.STOP_HTTP_SERVER)
	// 	stoppedMsg := stopHttpServCh.Subscribe()
	// 	test.Log(stoppedMsg)
	// })()

	//START EVENT
	startHttpServCh = httpCh.New(subscriptions.START_HTTP_SERVER)
	startMsg := msgbus.NewMessage("localhost:3000")
	startHttpServCh.Publish(startMsg)

	time.Sleep(20 * time.Second)
}
