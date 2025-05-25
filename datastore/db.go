package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const bufSize = 8192
const outFileName = "current-data"

var ErrNotFound = fmt.Errorf("record does not exist")

type hashInd map[string]int64

type FileSegment struct {
	index   hashInd
	outPath string
	mutex   sync.RWMutex
}

type Db struct {
	out         *os.File
	outOffset   int64
	dir         string
	segmentSize int64
	totalNumber int
	segments    []*FileSegment
	indexMutex  sync.RWMutex
}

func (s *FileSegment) getValue(position int64) (string, error) {
	file, err := os.Open(s.outPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err = file.Seek(position, io.SeekStart); err != nil {
		return "", err
	}
	reader := bufio.NewReader(file)

	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return "", err
	}
	size := binary.LittleEndian.Uint32(header)
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return "", err
	}

	buf := append(header, data...)
	var e entry
	e.Decode(buf)
	if e.isDeleted {
		return "", ErrNotFound
	}
	return e.value, nil
}

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments:    make([]*FileSegment, 0),
		dir:         dir,
		segmentSize: segmentSize,
	}

	if err := db.newSegment(); err != nil {
		return nil, err
	}
	if err := db.recover(); err != nil && err != io.EOF {
		return nil, err
	}
	return db, nil
}

func (db *Db) newSegment() error {
	outFile := fmt.Sprintf("%s%d", outFileName, db.totalNumber)
	outPath := filepath.Join(db.dir, outFile)
	db.totalNumber++

	f, err := os.OpenFile(outPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	if db.out != nil {
		db.out.Close()
	}
	db.out = f
	db.outOffset = 0

	newSeg := &FileSegment{outPath: outPath, index: make(hashInd)}
	db.segments = append(db.segments, newSeg)
	if len(db.segments) >= 3 {
		db.consolidateSegments()
	}
	return nil
}

func (db *Db) consolidateSegments() {
	go func() {
		outFile := fmt.Sprintf("%s%d", outFileName, db.totalNumber)
		outPath := filepath.Join(db.dir, outFile)
		db.totalNumber++

		newSeg := &FileSegment{outPath: outPath, index: make(hashInd)}
		f, err := os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			return
		}
		defer f.Close()
		var offset int64

		last := len(db.segments) - 2
		for i := 0; i <= last; i++ {
			s := db.segments[i]
			for key, pos := range s.index {
				if i < last {
					skip := false
					for _, seg := range db.segments[i+1 : last+1] {
						if _, ok := seg.index[key]; ok {
							skip = true
							break
						}
					}
					if skip {
						continue
					}
				}
				val, err := s.getValue(pos)
				if err == ErrNotFound {
					continue
				}
				entry := entry{key: key, value: val}
				n, err := f.Write(entry.Encode())
				if err == nil {
					newSeg.index[key] = offset
					offset += int64(n)
				}
			}
		}
		db.segments = []*FileSegment{newSeg, db.segments[len(db.segments)-1]}
	}()
}

func (db *Db) recover() error {
	in := bufio.NewReaderSize(db.out, bufSize)
	var offset int64
	for {
		header, err := in.Peek(4)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(header)
		data := make([]byte, size)
		n, err := in.Read(data)
		if err != nil {
			return err
		}
		buf := append(header, data[:n]...)
		var e entry
		e.Decode(buf)
		if !e.isDeleted {
			db.segments[len(db.segments)-1].index[e.key] = offset
		}
		offset += int64(n)
	}
	return nil
}

func (db *Db) Get(key string) (string, error) {
	db.indexMutex.RLock()
	defer db.indexMutex.RUnlock()
	for i := len(db.segments) - 1; i >= 0; i-- {
		seg := db.segments[i]
		seg.mutex.RLock()
		pos, ok := seg.index[key]
		seg.mutex.RUnlock()
		if ok {
			return seg.getValue(pos)
		}
	}
	return "", ErrNotFound
}

func (db *Db) Put(key, value string) error {
	entry := entry{key: key, value: value}
	db.indexMutex.Lock()
	defer db.indexMutex.Unlock()

	enc := entry.Encode()
	sz := int64(len(enc))
	if st, err := db.out.Stat(); err != nil {
		return err
	} else if st.Size()+sz > db.segmentSize {
		if err := db.newSegment(); err != nil {
			return err
		}
	}
	n, err := db.out.Write(enc)
	if err != nil {
		return err
	}
	seg := db.segments[len(db.segments)-1]
	seg.mutex.Lock()
	delete(seg.index, key)
	seg.index[key] = db.outOffset
	seg.mutex.Unlock()
	db.outOffset += int64(n)
	return nil
}

func (db *Db) Delete(key string) error {
	db.indexMutex.Lock()
	defer db.indexMutex.Unlock()
	entry := entry{key: key, value: "", isDeleted: true}
	enc := entry.Encode()
	sz := int64(len(enc))
	if st, err := db.out.Stat(); err != nil {
		return err
	} else if st.Size()+sz > db.segmentSize {
		if err := db.newSegment(); err != nil {
			return err
		}
	}
	n, err := db.out.Write(enc)
	if err != nil {
		return err
	}
	seg := db.segments[len(db.segments)-1]
	seg.mutex.Lock()
	delete(seg.index, key)
	seg.mutex.Unlock()
	db.outOffset += int64(n)
	return nil
}

func (db *Db) Close() { db.out.Close() }
