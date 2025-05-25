package datastore

import (
	"encoding/binary"
)

type entry struct {
	key       string
	value     string
	isDeleted bool
}

func (e *entry) Encode() []byte {
	kl := len(e.key)
	vl := len(e.value)
	size := 4 + 1 + 4 + kl + 4 + vl
	buf := make([]byte, size)
	// total size
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size-4))
	// flag
	if e.isDeleted {
		buf[4] = 1
	} else {
		buf[4] = 0
	}
	// key length
	binary.LittleEndian.PutUint32(buf[5:9], uint32(kl))
	copy(buf[9:9+kl], e.key)
	// value length
	off := 9 + kl
	binary.LittleEndian.PutUint32(buf[off:off+4], uint32(vl))
	copy(buf[off+4:off+4+vl], e.value)
	return buf
}

func (e *entry) Decode(input []byte) {
	// flag
	e.isDeleted = input[4] == 1
	// key length
	kl := int(binary.LittleEndian.Uint32(input[5:9]))
	e.key = string(input[9 : 9+kl])
	// value length
	off := 9 + kl
	vl := int(binary.LittleEndian.Uint32(input[off : off+4]))
	e.value = string(input[off+4 : off+4+vl])
}
