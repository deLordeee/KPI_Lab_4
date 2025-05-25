package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"
const teamName = "winners" 

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
		
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, teamName))
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Request %d: expected status 200, got %d", i, resp.StatusCode)
		}

		
		from := resp.Header.Get("lb-from")
		t.Logf("Response %d from [%s]", i, from)
		if from == "" {
			t.Error("Missing 'lb-from' header in response")
		}
		seenBackends[from]++

	
		var data map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("Request %d: failed to decode response: %v", i, err)
		}

	
		if data["key"] != teamName {
			t.Errorf("Request %d: expected key '%s', got '%s'", i, teamName, data["key"])
		}
		if data["value"] == "" {
			t.Errorf("Request %d: expected non-empty value, got empty", i)
		}

		t.Logf("Request %d: received data %+v", i, data)
		resp.Body.Close()
	}

	if len(seenBackends) < 2 {
		t.Error("Balancer did not distribute requests among multiple backends")
	}

	t.Logf("Requests distributed across backends: %+v", seenBackends)
}

func TestBalancerNotFound(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=nonexistent", baseAddress))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404 for non-existent key, got %d", resp.StatusCode)
	}

	t.Log("Correctly returned 404 for non-existent key")
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration benchmark is not enabled")
	}

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, teamName))
		if err != nil {
			b.Fatalf("Benchmark request failed: %v", err)
		}
		_ = resp.Body.Close()
	}
}
