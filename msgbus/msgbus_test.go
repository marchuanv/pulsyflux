package msgbus

import (
	"bytes"
	"net/http"
	"pulsyflux/message"
	"testing"
)

func TestMessageBus(t *testing.T) {
	msgBus := New(3000)
	msgBus.Start()

	msg, err := message.New("Hello World")
	if err != nil {
		t.Errorf("CtorError: failed to create message")
		return
	}

	// HTTP endpoint
	posturl := "http://localhost:3000/subscriptions/testchannel"

	// JSON body
	serialisedMsg, err := msg.Serialise()
	if err != nil {
		t.Errorf("CtorError: failed to serialise message")
		return
	}

	body := []byte(serialisedMsg)

	// Create a HTTP post request
	res, err := http.Post(posturl, "text/plain", bytes.NewBuffer(body))
	if err != nil {
		t.Error(err)
		return
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Errorf("Response StatusCode:%d, StatusMessage:%s", res.StatusCode, res.Status)
		return
	} else {
		msg := msgBus.Dequeue()
		if msg == nil {
			t.Errorf("MessageBus: expected a message to be dequeued")
		} else {
			t.Errorf("MessageId:%s, MessageText:%s", msg.Id, msg.Text)
		}
	}
	// time.Sleep(10 * time.Second)
}
