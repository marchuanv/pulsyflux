package events

import (
	"testing"
	"time"
)

type GlobalMsg struct{}
type LocalMsg struct{}

func TestEvent(test *testing.T) {
	HelloWorldEvent := Event[*GlobalMsg, *LocalMsg]("100447a1-bf04-4052-b6a3-43061532b1ef")
	eventRaised := false
	expectedGlobalMsg := &GlobalMsg{}
	expectedLocalMsg := &LocalMsg{}
	HelloWorldEvent.Subscribe(func(localMsg *LocalMsg) {
		eventRaised = true
		if expectedLocalMsg != localMsg {
			test.Fail()
		}
	})
	HelloWorldEvent.Convert(func(globalMsg *GlobalMsg) *LocalMsg {
		if expectedGlobalMsg != globalMsg {
			test.Fail()
		}
		return expectedLocalMsg
	})
	HelloWorldEvent.Publish(expectedGlobalMsg)
	time.Sleep(1000 * time.Millisecond)
	if !eventRaised {
		test.Fail()
	}
}
