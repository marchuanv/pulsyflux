package msgbus

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type MessageBus struct {
	mu sync.Mutex
}

func New(port int) (*MessageBus, error) {
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        new(MessageBus),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err := httpServer.ListenAndServe()
	return nil, err
}

func (h *MessageBus) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
}
