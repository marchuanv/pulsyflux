package connect

import (
	"fmt"
	"io"
	"net/http"
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
	"pulsyflux/task"
	"pulsyflux/util"
)

type HttpRequest struct {
	Schema HttpSchema
	Method HttpMethod
	Host   string
	Port   int
	Path   string
	Data   string
}

type HttpResponse struct {
	Request       HttpRequest
	StatusCode    int
	StatusMessage string
	Data          string
}

func HttpRequestSubscriptions() {
	task.DoNow(msgbus.New(subscriptions.HTTP), func(httpCh *msgbus.Channel) any {

		reqBody := task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) io.Reader {
			httpRequestBodyCh := httpReqCh.New(subscriptions.HTTP_REQUEST_DATA)
			requestBodyMsg := httpRequestBodyCh.Subscribe()
			reqBody := util.ReaderFromString(requestBodyMsg.String())
			return reqBody
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST_DATA)
		})

		task.DoNow(httpCh.New(subscriptions.HTTP_REQUEST), func(httpReqCh *msgbus.Channel) any {
			httpReqCh.Subscribe()
			req, err := http.NewRequest(httpMethod, requestURL.String(), reqBody)
			if err != nil {
				panic(err)
			}
			res, err := http.DefaultClient.Do(req)
			req.Body.Close()
			req = nil
			if err != nil {
				panic(err)
			}
			resBody := util.StringFromReader(res.Body)
			resBodyMsg := msgbus.NewMessage(resBody)
			httpReqCh.Publish(resBodyMsg)
			return ""
		}, func(err error, httpReqCh *msgbus.Channel) *msgbus.Channel {
			return httpReqCh.New(subscriptions.INVALID_HTTP_REQUEST)
		})

		return nil
	}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
		task.DoNow(errorPub, func(errorPub *msgbus.Channel) any {
			errorMsg := msgbus.NewMessage(err.Error())
			errorPub.Publish(errorMsg)
			return nil
		}, func(err error, errorPub *msgbus.Channel) *msgbus.Channel {
			fmt.Println("MessagePublishFail: could not publish the error message to the channel, logging the error here: ", err)
			return errorPub
		})
		return errorPub
	})
}
