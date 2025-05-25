package main

import (
  "encoding/json"
  "flag"
  "github.com/roman-mazur/architecture-practice-4-template/datastore"
  "github.com/roman-mazur/architecture-practice-4-template/httptools"
  "github.com/roman-mazur/architecture-practice-4-template/signal"
  "io/ioutil"
  "log"
  "net/http"
)

var port = flag.Int("port", 8083, "server port")

func main() {

  dir, err := ioutil.TempDir("", "temp-dir")
  if err != nil {
    log.Fatal(err)
  }

  db, err := datastore.NewDb(dir, 250)
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  h := http.NewServeMux()

  h.HandleFunc("/db/", func(rw http.ResponseWriter, req *http.Request) {
    key := req.URL.Path[4:]

    switch req.Method {
    case http.MethodGet:
      value, err := db.Get(key)
      if err != nil {
        rw.WriteHeader(http.StatusNotFound)
        json.NewEncoder(rw).Encode(map[string]string{"error": "Not found"})
        return
      }

      resp := struct {
        Key   string json:"key"
        Value string json:"value"
      }{Key: key, Value: value}

      rw.Header().Set("Content-Type", "application/json")
      rw.WriteHeader(http.StatusOK)
      json.NewEncoder(rw).Encode(resp)

    case http.MethodPost:
      var body struct {
        Value string json:"value"
      }
      if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
        rw.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(rw).Encode(map[string]string{"error": "Bad request"})
        return
      }

      if err := db.Put(key, body.Value); err != nil {
        rw.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(rw).Encode(map[string]string{"error": "Internal Server Error"})
        return
      }
      rw.WriteHeader(http.StatusCreated)

    default:
      rw.WriteHeader(http.StatusBadRequest)
      json.NewEncoder(rw).Encode(map[string]string{"error": "Method not allowed"})
    }
  })

  server := httptools.CreateServer(*port, h)
  server.Start()
  signal.WaitForTerminationSignal()
}