package tsar

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
)

const PointerSize = 4

type Entry struct {
	Key      string
	Pointers []uint32
}

func marshalNumPointers(e *Entry) []byte {
	if len(e.Pointers) <= math.MaxUint16 {
		return bytesOfUint16(uint16(len(e.Pointers)))
	} else {
		return append([]byte{0, 0}, bytesOfUint32(uint32(len(e.Pointers)))...)
	}
}

func marshalEntry(e *Entry) []byte {
	var res []byte
	res = append(res, uint8(len(e.Key)))
	res = append(res, marshalNumPointers(e)...)
	res = append(res, []byte(e.Key)...)
	for _, p := range e.Pointers {
		res = append(res, bytesOfUint32(p)...)
	}
	return res
}

func unmarshalEntryReader(r io.Reader) (*Entry, error) {
	// key length
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("when reading key length byte: %w", err)
	}
	keyLen := int(buf[0])

	// num checkpoints
	buf = make([]byte, 2)
	_, err = r.Read(buf)
	numPtrs := int(uint16OfBytes(buf))
	if numPtrs == 0 && err == nil {
		buf = make([]byte, 4)
		_, err = r.Read(buf)
		numPtrs = int(uint32OfBytes(buf))
	}
	if err != nil {
		return nil, fmt.Errorf("when reading %d key num checkpoint bytes: %w", len(buf), err)
	}

	// key
	buf = make([]byte, keyLen)
	_, err = r.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("when reading %d key bytes: %w", len(buf), err)
	}
	key := string(buf)

	// checkpoints
	buf = make([]byte, numPtrs*PointerSize)
	_, err = r.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("when reading %d checkpoints bytes: %w", len(buf), err)
	}
	pointers := make([]uint32, numPtrs)
	for i := 0; i < numPtrs; i++ {
		j := i * PointerSize
		pointers[i] = uint32OfBytes(buf[j : j+PointerSize])
	}

	e := &Entry{
		Key:      key,
		Pointers: pointers,
	}
	return e, nil
}

func unmarshalEntry(entryBytes []byte) (*Entry, error) {
	return unmarshalEntryReader(bytes.NewReader(entryBytes))
}

func (e *Entry) length() uint32 {
	return uint32(1 + len(marshalNumPointers(e)) + len(e.Key) + len(e.Pointers)*PointerSize)
}

func NewEntryList() EntryList {
	return make(map[string][]uint32)
}

type EntryList map[string][]uint32

func (l EntryList) Append(key string, ptr uint32) error {
	if len(key) > MaxKeyLen {
		return errors.New("Key must be smaller than 256 bytes")
	}

	ptrs := l[key]

	if len(ptrs) >= MaxEntryPointers {
		return fmt.Errorf("reached maximum number of checkpoints (%d) for key %s", MaxEntryPointers, key)
	}

	l[key] = append(ptrs, ptr)
	return nil
}

func (l EntryList) Set(key string, pointers []uint32) error {
	if len(key) > MaxKeyLen {
		return errors.New("Key must be smaller than 256 bytes")
	}
	if len(pointers) > math.MaxUint32 {
		return fmt.Errorf("value contains over %d items", math.MaxUint32)
	}
	l[key] = pointers
	return nil
}

func (l EntryList) Remove(key string) {
	delete(l, key)
}

func (l EntryList) ToIndex() *Index {
	var keys []string
	for key, pointers := range l {
		if len(pointers) > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var entries []*Entry
	offsets := make(map[*Entry]uint32)
	var offset uint32 = 0
	for _, key := range keys {
		e := &Entry{
			Key:      key,
			Pointers: l[key],
		}
		entries = append(entries, e)
		offsets[e] = offset
		offset += e.length()
	}

	checkpoints := func(lo, hi int) (res []uint32) {
		for i := lo; i < hi; i += PartitionSize {
			res = append(res, offsets[entries[i]])
		}
		res = append(res, offsets[entries[hi]])
		return res
	}(0, len(entries)-1)

	return &Index{
		checkpoints: checkpoints,
		entries:     entries,
	}
}
