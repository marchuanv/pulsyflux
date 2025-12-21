package httpcontainer

type maxHeaderBytes int

func newHttpMaxHeaderBytes(code int) maxHeaderBytes {
	return maxHeaderBytes(code)
}

func newDefaultHttpMaxHeaderBytes() maxHeaderBytes {
	return maxHeaderBytes(1024 * 1024) // 1MB
}
