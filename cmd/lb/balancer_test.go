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
