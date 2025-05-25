package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
	"log"
	"net/http"
	"os"
	"time"
)

var port = flag.Int("port", 8080, "server port")

const dbUrl = "http://db:8083"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func main() {
	// Try to save current date with retries
	saveCurrentDateWithRetry(dbUrl, "winners", 10, 2*time.Second)
	
	http.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	http.HandleFunc("/api/v1/some-data", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "Key required", http.StatusBadRequest)
			return
		}

		resp, err := http.Get(dbUrl + "/db/" + key)
		if err != nil {
			http.Error(w, "Service is not available", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			http.NotFound(w, r)
			return
		}

		var data map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			http.Error(w, "Error while decoding response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	server := httptools.CreateServer(*port, nil)
	server.Start()
	time.Sleep(5 * time.Second)
	signal.WaitForTerminationSignal()
}

func saveCurrentDateWithRetry(dbURL, teamKey string, maxRetries int, delay time.Duration) {
	for i := 0; i < maxRetries; i++ {
		if err := saveCurrentDate(dbURL, teamKey); err != nil {
			log.Printf("Attempt %d failed to save current date: %v", i+1, err)
			if i < maxRetries-1 {
				log.Printf("Retrying in %v...", delay)
				time.Sleep(delay)
				continue
			}
			log.Printf("Failed to save current date after %d attempts, continuing without it", maxRetries)
		} else {
			log.Println("Successfully saved current date to database")
			return
		}
	}
}

func saveCurrentDate(dbURL, teamKey string) error {
	currentDate := time.Now().Format("2006-01-02")
	data := map[string]string{
		"value": currentDate,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = http.Post(dbURL+"/db/"+teamKey, "application/json", bytes.NewBuffer(jsonData))
	return err
}
