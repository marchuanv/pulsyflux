package msgsub

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"pulsyflux/message"
	"strconv"
	"time"
)

type Address struct {
	hostName string
	port     int
}
type Subscription struct {
	channel string
	from    Address
	to      Address
}

func New(channel string, fromAddress string, toAddress string) (*Subscription, error) {

	if len(channel) == 0 {
		return nil, errors.New("The channel argument is an empty string.")
	}

	fromHost, fromPort, fromErr := getHostAndPortFromAddress(fromAddress)
	if fromErr != nil {
		return nil, fromErr
	}

	toHost, toPort, toErr := getHostAndPortFromAddress(toAddress)
	if toErr != nil {
		return nil, toErr
	}

	subscription := Subscription{
		channel,
		Address{fromHost, fromPort}, //from
		Address{toHost, toPort},     //to
	}

	return &subscription, nil
}
func (sub *Subscription) subscribe() error {

	serialisedSub, err := serialiseSubscription(*sub)
	if err != nil {
		return err
	}

	msg, err := message.New(serialisedSub)
	if err != nil {
		return err
	}

	// HTTP endpoint
	posturl := fmt.Sprintf("http://%s:%s/subscriptions/%s", sub.to.hostName, sub.to.port, sub.channel)
	posturl = url.PathEscape(posturl)

	// JSON body
	serialisedMsg, err := msg.Serialise()
	if err != nil {
		return err
	}

	body := []byte(serialisedMsg)

	// Create a HTTP post request
	res, err := http.Post(posturl, "text/plain", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Response StatusCode:%s, StatusMessage:%s", res.StatusCode, res.Status))
	}
	return nil
}
func (sub *Subscription) receive() (*message.Message, error) {
	msg := message.Message{}
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%s", sub.from.port),
		Handler:        myHandler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err := httpServer.ListenAndServe()
	return &msg, err
}

func getHostAndPortFromAddress(address string) (string, int, error) {
	hostStr, portStr, hostPortErr := net.SplitHostPort(address)
	if hostPortErr != nil {
		return "", 0, hostPortErr
	}
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil {
		return "", 0, convErr
	}
	return hostStr, port, nil
}

// serialise subscription
func serialiseSubscription(m Subscription) (string, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// deserialise subscription
func deserialiseSubscription(subStr string) (Subscription, error) {
	sub := Subscription{}
	by, err := base64.StdEncoding.DecodeString(subStr)
	if err != nil {
		return sub, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&sub)
	if err != nil {
		return sub, err
	}
	return sub, nil
}
