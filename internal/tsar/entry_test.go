package tsar

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

var randString = func(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestEntryMarshaling(t *testing.T) {

	for i := 0; i < 250; i++ {
		e := &Entry{Key: randString((i + 5) / 2)}
		for j := 0; j < (i+5)*3; j++ {
			e.Pointers = append(e.Pointers, rand.Uint32())
		}

		e1, err := unmarshalEntry(marshalEntry(e))
		if err != nil {
			t.Fatal(err)
		}

		if e.Key != e1.Key {
			t.Fatal("expected", e.Key, "got", e1.Key)
		}

		if len(e.Pointers) != len(e1.Pointers) {
			t.Fatal("expected Pointers len", len(e.Pointers), "got", len(e1.Pointers))
		}
		for i, v := range e.Pointers {
			if v != e1.Pointers[i] {
				t.Fatal("expected Pointers", v, "at position", i, "got", e.Pointers[i])
			}
		}

		r := bytes.NewReader(marshalEntry(e))
		e1, err = unmarshalEntryReader(r)
		if err != nil {
			t.Fatal("got error,", err)
		}

		if e.Key != e1.Key {
			t.Fatal("expected", e.Key, "got", e1.Key)
		}

		if len(e.Pointers) != len(e1.Pointers) {
			t.Fatal("expected Pointers len", len(e.Pointers), "got", len(e1.Pointers))
		}
		for i, v := range e.Pointers {
			if v != e1.Pointers[i] {
				t.Fatal("expected Pointers", v, "at position", i, "got", e.Pointers[i])
			}
		}
	}

}
