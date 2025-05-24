package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupServers(count int) []*httptest.Server {
	var servers []*httptest.Server
	for i := 0; i < count; i++ {
		si := i
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			fmt.Fprintf(w, "server-%d", si)
		}))
		servers = append(servers, ts)
	}
	return servers
}

func TestSelectBackendLeastTraffic(t *testing.T) {
	
	servers := setupServers(3)
	defer func() {
		for _, ts := range servers {
			ts.Close()
		}
	}()


	serversPool = []string{}
	trafficCounters = make(map[string]*uint64)
	for _, ts := range servers {
		addr := ts.Listener.Addr().String()
		serversPool = append(serversPool, addr)
		trafficCounters[addr] = new(uint64)
	}

	
	atomic.AddUint64(trafficCounters[serversPool[0]], 500)
	atomic.AddUint64(trafficCounters[serversPool[1]], 100)
	atomic.AddUint64(trafficCounters[serversPool[2]], 200)

	
	chosen := selectBackend()
	assert.Equal(t, serversPool[1], chosen)
}

func TestForwardUpdatesTraffic(t *testing.T) {
	
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	}))
	defer ts.Close()

	addr := ts.Listener.Addr().String()
	
	serversPool = []string{addr}
	trafficCounters = map[string]*uint64{addr: new(uint64)}

	req := httptest.NewRequest("GET", "http://ignored", nil)
	rw := httptest.NewRecorder()

	err := forward(addr, rw, req)
	assert.NoError(t, err)

	cnt := atomic.LoadUint64(trafficCounters[addr])
	assert.Equal(t, uint64(len("hello world")), cnt)
}
