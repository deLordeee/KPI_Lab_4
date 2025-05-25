package datastore

import "testing"

func TestEntryEncodeDecode(t *testing.T) {
	e := entry{key: "a", value: "b", isDeleted: false}
	data := e.Encode()
	var e2 entry
	e2.Decode(data)
	if e2.key != "a" || e2.value != "b" || e2.isDeleted {
		t.Error("Entry encode/decode failed")
	}
	tomb := entry{key: "x", value: "", isDeleted: true}
	data = tomb.Encode()
	var t2 entry
	t2.Decode(data)
	if t2.key != "x" || !t2.isDeleted {
		t.Error("Tombstone encode/decode failed")
	}
}
