package tsar

import (
	"log"
	"math"
	"math/rand"
	"testing"
	"time"
)

func createTestIndex(size int) *Index {
	rand.Seed(time.Now().UnixNano())
	var list = NewEntryList()
	for i := 0; i < size; i++ {
		key := randString(rand.Intn(230) + 5)
		var ptrs []uint32

		for j := 0; j < rand.Intn(40)+1; j++ {
			ptrs = append(ptrs, rand.Uint32())

		}

		if i%size/4 == 0 {
			for j := 0; j < 2*math.MaxUint16; j++ {
				ptrs = append(ptrs, rand.Uint32())
			}
		}

		err := list.Set(key, ptrs)
		if err != nil {
			log.Fatal("err", err)
		}
	}
	return list.ToIndex()
}

func TestIndexMarshaling(t *testing.T) {

	var list = NewEntryList()

	for i := 0; i < 1000; i++ {
		key := randString(rand.Intn(230) + 5)
		var ptrs []uint32

		for j := 0; j < rand.Intn(40)+1; j++ {
			ptrs = append(ptrs, rand.Uint32())
		}

		err := list.Set(key, ptrs)
		if err != nil {
			t.Fatal("err", err)
		}
	}

	i1 := list.ToIndex()

	i2, err := UnmarshalIndex(MarshalIndex(i1))
	if err != nil {
		t.Fatal("error", err)
	}

	if len(i2.checkpoints) != len(i1.checkpoints) {
		t.Fatal("expected number of checkpoint to be ", len(i1.checkpoints), ", got ", len(i2.checkpoints))
	}

	for i, p1 := range i1.checkpoints {
		p2 := i2.checkpoints[i]
		if p1 != p2 {
			t.Fatal("expected point ", i, " offset to be ", p1, " got ", p2)
		}
	}

	if len(i2.entries) != len(i1.entries) {
		t.Fatal("expected number of entries to be ", len(i1.entries), ", got ", len(i2.entries))
	}

	for i, e1 := range i1.entries {
		e2 := i2.entries[i]
		if e1.Key != e2.Key {
			t.Fatal("expected entry Key ", e1.Key, ", got ", e2.Key, " @ ", i)
		}
		if len(e1.Pointers) != len(e2.Pointers) {
			t.Fatal("expected number of values to be ", len(e1.Pointers), ", got ", len(e1.Pointers))
		}
		for j, v1 := range e1.Pointers {
			v2 := e2.Pointers[j]
			if v2 != v1 {
				t.Fatal("expected Pointers ", v1, ", got ", v2, " @ ", i, " : ", j)
			}
		}
	}
}

func testIndexFind(source, test *Index, t *testing.T) {
	for j, e1 := range source.entries {
		list, err := test.Find(e1.Key, MatchEqual)
		if err != nil {
			t.Fatal("err, ", err)
		}
		if len(list) != 1 {
			//t.Fatal("expected result of 1, got ", len(list), " @ ", j , "  ", fmt.Sprintf("\n%v\n%+v\n%+v\n",e1, list[0], list[1]))
			t.Fatal("expected result of 1, got ", len(list), " @ ", j)
		}
		e2 := list[0]

		if e1.Key != e2.Key {
			t.Fatal("expected result of list Key to be ", e1.Key, ", got ", e2.Key, " @ ", j)
		}

		if len(e1.Pointers) != len(e2.Pointers) {
			t.Fatal("expected values to contain ", len(e1.Pointers), " elements, got ", len(e2.Pointers), " @ ", j)
		}

		for k, v1 := range e1.Pointers {
			v2 := e2.Pointers[k]
			if v1 != v2 {
				t.Fatal("expected Pointers to be ", v1, ", got ", v2, " @ ", j, " : ", k)
			}
		}

	}
}

func TestIndexFind(t *testing.T) {

	var list = NewEntryList()

	for i := 0; i < 1013; i++ {
		key := randString(rand.Intn(230) + 5)
		var ptrs []uint32

		for j := 0; j < rand.Intn(40)+1; j++ {
			ptrs = append(ptrs, rand.Uint32())
		}

		err := list.Set(key, ptrs)
		if err != nil {
			t.Fatal("err", err)
		}
	}

	i := list.ToIndex()

	testIndexFind(i, i, t)
}

func TestIndexFindLazy(t *testing.T) {

	var list = NewEntryList()

	for i := 0; i < 1013; i++ {
		key := randString(rand.Intn(230) + 5)
		var ptrs []uint32
		for j := 0; j < rand.Intn(40)+1; j++ {
			ptrs = append(ptrs, rand.Uint32())
		}

		err := list.Set(key, ptrs)
		if err != nil {
			t.Fatal("err", err)
		}
	}

	i1 := list.ToIndex()
	i2, err := UnmarshalIndexLazy(MarshalIndex(i1))
	if err != nil {
		t.Fatal("err, ", err)
	}

	testIndexFind(i1, i2, t)
}

var needles []*Entry
var testIndex *Index
var testIndexRaw []byte
var lazyTestIndex *Index

func init() {
	var err error
	testIndex = createTestIndex(500013)
	needles = testIndex.entries
	testIndexRaw = MarshalIndex(testIndex)
	lazyTestIndex, err = UnmarshalIndexLazy(testIndexRaw)
	if err != nil {
		panic(err)
	}
}

var voidres []*Entry

func BenchmarkLoadedIndexOptimal(b *testing.B) {
	var r []*Entry
	for n := 0; n < b.N; n++ {
		needle := n % len(needles)
		res, err := testIndex.Find(needles[needle].Key, MatchEqual)
		if err != nil {
			b.Fatal("err, ", err)
		}
		r = res
	}
	voidres = r
}
func BenchmarkLazyIndexOptimal(b *testing.B) {
	var r []*Entry
	for n := 0; n < b.N; n++ {
		needle := n % len(needles)
		res, err := lazyTestIndex.Find(needles[needle].Key, MatchEqual)
		if err != nil {
			b.Fatal("err, ", err)
		}
		r = res
	}
	voidres = r
}

func BenchmarkLoadedIndexUnmarshal(b *testing.B) {
	var r []*Entry
	for n := 0; n < b.N; n++ {
		needle := n % len(needles)
		index, err := UnmarshalIndex(testIndexRaw)
		if err != nil {
			b.Fatal("err, ", err)
		}
		res, err := index.Find(needles[needle].Key, MatchEqual)
		if err != nil {
			b.Fatal("err, ", err)
		}
		r = res
	}
	voidres = r
}
func BenchmarkLazyIndexUnmarshal(b *testing.B) {
	var r []*Entry
	for n := 0; n < b.N; n++ {
		needle := n % len(needles)
		index, err := UnmarshalIndexLazy(testIndexRaw)
		if err != nil {
			b.Fatal("err, ", err)
		}
		res, err := index.Find(needles[needle].Key, MatchEqual)
		if err != nil {
			b.Fatal("err, ", err)
		}
		r = res
	}
	voidres = r
}
