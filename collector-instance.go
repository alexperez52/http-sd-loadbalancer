package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

// Collector holds the data acout a server
type Collector struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// SetAlive for this Collector
func (c *Collector) SetAlive(alive bool) {
	c.mux.Lock()
	c.Alive = alive
	c.mux.Unlock()
}

// IsAlive returns true when Collector is alive
func (c *Collector) IsAlive() (alive bool) {
	c.mux.RLock()
	alive = c.Alive
	c.mux.RUnlock()
	return
}
