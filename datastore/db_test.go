package datastore

import (
  "io/ioutil"
  "os"

  "testing"
)

type Data struct { key, value string }

func TestPutGetDelete(t *testing.T) {
  dir, err := ioutil.TempDir("", "test-db")
  if err != nil { t.Fatal(err) }
  defer os.RemoveAll(dir)

  db, err := NewDb(dir, 100)
  if err != nil { t.Fatal(err) }
  defer db.Close()

  data := []Data{{"k1", "v1"}, {"k2", "v2"}, {"k3", "v3"}}
  for _, d := range data {
    if err := db.Put(d.key, d.value); err != nil {
      t.Errorf("Put error: %v", err)
    }
    if got, err := db.Get(d.key); err != nil || got != d.value {
      t.Errorf("Get = %v, %v; want %v", got, err, d.value)
    }
  }
  db.Delete("k2")
  if _, err := db.Get("k2"); err != ErrNotFound {
    t.Errorf("Deleted key found: %v", err)
  }
}
