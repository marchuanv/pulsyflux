package socket

import "sync"

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, maxFrameSize)
		return &buf
	},
}

func getBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

func putBuffer(buf *[]byte) {
	bufferPool.Put(buf)
}
