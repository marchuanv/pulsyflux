package msgsub

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"pulsyflux/message"
	"pulsyflux/msgbus"
)

type Subscription struct {
	message *message.Message
	bus     *msgbus.MessageBus
}

func NewSubscription(msg *message.Message) (*Subscription, error) {
	fromAddress := msg.FromAddress()
	bus, err := msgbus.New(fromAddress.Host, fromAddress.Port)
	if err != nil {
		return nil, err
	}
	subscription := Subscription{msg, bus}
	return &subscription, nil
}

func (subscription *Subscription) Send() error {
	serialised, err := subscription.message.Serialise()
	if err != nil {
		return err
	}
	body := []byte(serialised)
	toAddress := subscription.message.ToAddress()
	channel := subscription.message.Channel()
	url := fmt.Sprintf("http://%s:%d/%s/subscribe", toAddress.Host, toAddress.Port, channel)
	res, err := http.Post(url, "text/plain", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		errMsg := fmt.Sprintf("Response StatusCode:%d, StatusMessage:%s", res.StatusCode, res.Status)
		return errors.New(errMsg)
	}
	return nil
}

func (subscription *Subscription) Receive() *message.Message {
	msg := subscription.bus.Dequeue()
	return msg
}
