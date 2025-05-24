package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
  if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
    t.Skip("Integration test is not enabled")
  }

  const requestsCount = 10
  seenBackends := make(map[string]int)

  for i := 0; i < requestsCount; i++ {
    resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
    if err != nil {
      t.Fatalf("Request %d failed: %v", i, err)
    }
    from := resp.Header.Get("lb-from")
    t.Logf("Response %d from [%s]", i, from)

    if from == "" {
      t.Error("Missing 'lb-from' header in response")
    }

    seenBackends[from]++
    resp.Body.Close()
  }

  if len(seenBackends) < 2 {
    t.Error("Balancer did not distribute requests among multiple backends")
  }
}

func BenchmarkBalancer(b *testing.B) {
	// TODO: Реалізуйте інтеграційний бенчмарк для балансувальникка.
}
