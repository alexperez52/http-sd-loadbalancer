package main

import (
	"log"
	"net/url"
	"sync/atomic"
)

// ServerPool holds information about reachable backends
type ServerPool struct {
	collectors []*Collector
	current    uint64
}

// AddBackend to the server pool
func (s *ServerPool) AddBackend(backend *Collector) {
	s.collectors = append(s.collectors, backend)
}

// NextIndex atomically increase the counter and return an index
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.collectors)))
}

// MarkBackendStatus changes a status of a backend
func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, b := range s.collectors {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetNextPeer returns next active peer to take a connection
func (s *ServerPool) GetNextPeer() *Collector {
	// loop entire backends to find out an Alive backend
	next := s.NextIndex()
	l := len(s.collectors) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(s.collectors)     // take an index by modding
		if s.collectors[idx].IsAlive() { // if we have an alive backend, use it and store if its not the original one
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.collectors[idx]
		}
	}
	return nil
}

// HealthCheck pings the backends and update the status
func (s *ServerPool) HealthCheck() {
	for _, b := range s.collectors {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}
