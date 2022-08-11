package tsar

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

const MaxKeyLen = math.MaxUint8
const MaxEntryPointers = math.MaxUint32
const PartitionSize = 20
const CheckpointSize = 8 // bytes 0:4 unused (was row number), bytes 4:8 are entry offset

type Matcher func(candidate string, needle string) bool

var MatchEqual Matcher = func(a, b string) bool { return a == b }
var MatchPrefix Matcher = strings.HasPrefix

type Index struct {
	offset      int64
	reader      io.ReadSeeker
	checkpoints []uint32
	entries     []*Entry
}

func (i *Index) Find(needle string, match Matcher) ([]*Entry, error) {
	if i.reader == nil {
		lo, hi := 0, len(i.entries)
		for hi-lo > 1 {
			mid := (lo + hi) / 2
			e := i.entries[mid]
			if needle < e.Key {
				hi = mid
				continue
			}
			lo = mid
		}

		var res []*Entry
		for j := lo; j < len(i.entries) && match(i.entries[j].Key, needle); j++ {
			res = append(res, i.entries[j])
		}

		return res, nil
	}

	entryAtOffset := func(offset uint32) (e *Entry, err error) {
		seekOffset := i.offset + int64(offset)
		_, err = i.reader.Seek(seekOffset, 0)
		if err != nil {
			return nil, fmt.Errorf("when seeking offset %d", seekOffset)
		}
		return unmarshalEntryReader(i.reader)
	}

	var lo, hi uint32 = 0, uint32(len(i.checkpoints) - 1)
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		e, err := entryAtOffset(i.checkpoints[mid])
		if err != nil {
			return nil, err
		}
		if needle < e.Key {
			hi = mid
			continue
		}
		lo = mid
	}

	var res []*Entry
	lo, hi = i.checkpoints[lo], i.checkpoints[hi]
	last := i.checkpoints[len(i.checkpoints)-1]
	ok := false
	for lo <= hi || (ok && lo <= last) {
		e, err := entryAtOffset(lo)
		if err != nil {
			return nil, err
		}
		ok = match(e.Key, needle)
		if ok {
			res = append(res, e)
		}
		lo += e.length()
	}
	return res, nil
}

func MarshalIndex(i *Index) []byte {
	var buf = make([]byte, 0, i.checkpoints[len(i.checkpoints)-1])

	buf = append(buf, bytesOfUint32(uint32(len(i.checkpoints)))...)

	for _, p := range i.checkpoints {
		pBytes := append(bytesOfUint32(0), bytesOfUint32(p)...)
		buf = append(buf, pBytes...)
	}
	for _, e := range i.entries {
		buf = append(buf, marshalEntry(e)...)
	}

	return buf
}

func UnmarshalIndexLazyReader(reader io.ReadSeeker) (*Index, error) {
	numCheckpointsBytes := make([]byte, 4)
	_, err := reader.Read(numCheckpointsBytes)
	if err != nil {
		return nil, err
	}
	numCheckpoints := int(uint32OfBytes(numCheckpointsBytes))

	checkpointsBytes := make([]byte, numCheckpoints*CheckpointSize)
	_, err = reader.Read(checkpointsBytes)
	if err != nil {
		return nil, err
	}

	var checkpoints []uint32
	for j := 0; j < numCheckpoints; j++ {
		k := j * CheckpointSize
		p := uint32OfBytes(checkpointsBytes[k : k+CheckpointSize][4:8])
		checkpoints = append(checkpoints, p)
	}

	return &Index{
		offset:      int64(len(numCheckpointsBytes) + len(checkpointsBytes)),
		reader:      reader,
		checkpoints: checkpoints,
	}, nil
}

func UnmarshalIndexLazy(data []byte) (*Index, error) {
	return UnmarshalIndexLazyReader(newByteReadSeeker(data))
}

func UnmarshalIndex(data []byte) (*Index, error) {
	i, err := UnmarshalIndexLazy(data)
	if err != nil {
		return nil, err
	}

	for {
		e, err := unmarshalEntryReader(i.reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		i.entries = append(i.entries, e)
	}
	return i, nil
}
