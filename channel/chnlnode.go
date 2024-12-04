package channel

import (
	"pulsyflux/tree"
	"time"
)

type chnlNode struct {
	*tree.Node[Channel]
}

func NewChnlNode() Channel {
	chNode := tree.NewNode[Channel]()
	chNode.SetRef(NewChnl())
	return &chnlNode{chNode}
}

func (chNode *chnlNode) Read(receiveData func(data any)) {
	chLeafNodeReadable := chNode.GetLeafNode(func(ref Channel) bool { return ref.HasEvent(ChannelReadReady) })
	if chLeafNodeReadable != nil {
		chLeafNodeRefReadable := chLeafNodeReadable.Ref()
		chLeafNodeRefReadable.Read(receiveData)
		chNode.Read(receiveData)
	}
	chLeafNodeWriteReady := chNode.GetLeafNode(func(ref Channel) bool { return ref.HasEvent(ChannelWriteReady) })
	if chLeafNodeWriteReady != nil {
		time.Sleep(100 * time.Millisecond)
		chNode.Read(receiveData)
	}
}

func (chNode *chnlNode) Write(data any) {
	chNodeRef := chNode.Ref()
	if chNodeRef.HasEvent(ChannelReadReady) {
		chChildNode := NewChnlNode().(*chnlNode)
		chNode.AddChild(chChildNode.Node)
		chChildNode.Write(data)
		return
	}
	chNodeRef.Write(data)
}

func (chNode *chnlNode) RaiseEvent(event ChannelEvent) {
	chl := chNode.Ref()
	chl.RaiseEvent(event)
}

func (chNode *chnlNode) WaitForEvent(event ChannelEvent) {
	chl := chNode.Ref()
	chl.WaitForEvent(event)
}

func (chNode *chnlNode) HasEvent(event ChannelEvent) bool {
	chl := chNode.Ref()
	return chl.HasEvent(event)
}

func (chNode *chnlNode) Close() {
	chl := chNode.Ref()
	chl.Close()
}
