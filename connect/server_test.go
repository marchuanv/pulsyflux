package connect

import (
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"testing"
)

func TestServer(test *testing.T) {

	var err error
	var errorMsg msgbus.Msg
	var msg msgbus.Msg
	var httpServStartedCh *msgbus.Channel
	var startHttpServCh *msgbus.Channel
	var stopHttpServCh *msgbus.Channel
	var failedToStartHttpServCh *msgbus.Channel
	var failedToStopHttpServCh *msgbus.Channel
	var httpServPortUnavCh *msgbus.Channel
	var httpServerResCh *msgbus.Channel
	var httpServerSuccResCh *msgbus.Channel
	var httpServerErrResCh *msgbus.Channel

	httpCh, err := Subscribe()
	if err != nil {
		test.Fail()
	}

	startHttpServCh = httpCh.New(subscriptions.START_HTTP_SERVER)
	httpServStartedCh = httpCh.New(subscriptions.HTTP_SERVER_STARTED)
	failedToStartHttpServCh = startHttpServCh.New(subscriptions.FAILED_TO_START_HTTP_SERVER)
	stopHttpServCh = httpCh.New(subscriptions.STOP_HTTP_SERVER)
	failedToStopHttpServCh = stopHttpServCh.New(subscriptions.FAILED_TO_STOP_HTTP_SERVER)
	httpServPortUnavCh = startHttpServCh.New(subscriptions.FAILED_TO_LISTEN_ON_HTTP_SERVER_PORT)
	httpServerResCh = httpCh.New(subscriptions.HTTP_SERVER_RESPONSE)
	httpServerSuccResCh = httpServerResCh.New(subscriptions.HTTP_SERVER_SUCCESS_RESPONSE)
	httpServerErrResCh = httpServerResCh.New(subscriptions.HTTP_SERVER_ERROR_RESPONSE)

	startHttpServCh.Subscribe()

}
