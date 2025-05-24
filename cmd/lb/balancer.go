package main

import (
  "context"
  "flag"
  "fmt"
  "io"
  "log"
  "net/http"
  "sync/atomic"
  "time"

  "github.com/roman-mazur/architecture-practice-4-template/httptools"
  "github.com/roman-mazur/architecture-practice-4-template/signal"
)

var (
  port       = flag.Int("port", 8090, "load balancer port")
  timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
  https      = flag.Bool("https", false, "whether backends support HTTPs")

  traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")

  // initial list of backend addresses
  serversPool = []string{
    "server1:8080",
    "server2:8080",
    "server3:8080",
  }

  // traffic counters for each backend
  trafficCounters = make(map[string]*uint64)
)

func init() {
  // initialize counters
  for _, srv := range serversPool {
    trafficCounters[srv] = new(uint64)
  }
}

func scheme() string {
  if *https {
    return "https"
  }
  return "http"
}

// health reports whether the backend is healthy
func health(dst string) bool {
  ctx, _ := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
  req, _ := http.NewRequestWithContext(ctx, "GET",
    fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    return false
  }
  defer resp.Body.Close()
  return resp.StatusCode == http.StatusOK
}

// selectBackend picks the healthy backend with the least total response bytes
func selectBackend() string {
  var selected string
  var min uint64
  first := true
  for _, srv := range serversPool {
    if !health(srv) {
      continue
    }
    cnt := atomic.LoadUint64(trafficCounters[srv])
    if first || cnt < min {
      min = cnt
      selected = srv
      first = false
    }
  }
  if selected == "" {
    // fallback to first if none healthy
    return serversPool[0]
  }
  return selected
}

// forward proxies the request to dst and updates trafficCounters
func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
  ctx, _ := context.WithTimeout(r.Context(), time.Duration(*timeoutSec)*time.Second)
  fwdRequest := r.Clone(ctx)
  fwdRequest.RequestURI = ""
  fwdRequest.URL.Host = dst
  fwdRequest.URL.Scheme = scheme()
  fwdRequest.Host = dst

  resp, err := http.DefaultClient.Do(fwdRequest)
  if err != nil {
    log.Printf("Failed to get response from %s: %s", dst, err)
    rw.WriteHeader(http.StatusServiceUnavailable)
    return err
  }
  defer resp.Body.Close()

  // copy headers
  for k, values := range resp.Header {
    for _, v := range values {
      rw.Header().Add(k, v)
    }
  }
  if *traceEnabled {
    rw.Header().Set("lb-from", dst)
  }

  rw.WriteHeader(resp.StatusCode)
  // copy body and count bytes
  n, err := io.Copy(rw, resp.Body)
  if err != nil {
    log.Printf("Failed to write response body: %s", err)
  }
  // update traffic
  atomic.AddUint64(trafficCounters[dst], uint64(n))
  log.Printf("fwd %d from %s, bytes=%d", resp.StatusCode, dst, n)
  return nil
}

func main() {
  flag.Parse()

  // start health checks
  for _, server := range serversPool {
    srv := server
    go func() {
      for range time.Tick(10 * time.Second) {
        ok := health(srv)
        log.Println(srv, "healthy:", ok)
      }
    }()
  }

  frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
    // choose backend
    dst := selectBackend()
    forward(dst, rw, r)
  }))

  log.Println("Starting load balancer...")
  log.Printf("Tracing support enabled: %t", *traceEnabled)
  frontend.Start()
  signal.WaitForTerminationSignal()
}
