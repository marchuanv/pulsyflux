package socket

import "sync"

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, maxFrameSize)
		return &buf
	},
}

var framePool = sync.Pool{
	New: func() interface{} {
		return &frame{}
	},
}

var writeBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 8192+frameHeaderSize)
		return &buf
	},
}

func getBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

func putBuffer(buf *[]byte) {
	bufferPool.Put(buf)
}

func getFrame() *frame {
	return framePool.Get().(*frame)
}

func putFrame(f *frame) {
	f.Payload = nil
	framePool.Put(f)
}

func getWriteBuffer() *[]byte {
	return writeBufferPool.Get().(*[]byte)
}

func putWriteBuffer(buf *[]byte) {
	writeBufferPool.Put(buf)
}
