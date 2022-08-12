package mark

import (
	"bytes"
	"errors"
	"github.com/modfin/henry/slicez"
	"gopkg.in/yaml.v3"
	"time"
)

type Header struct {
	Title     string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Index struct {
	IdToNotes map[int]string   `json:"id_to_notes"`
	TagsToId  map[string][]int `json:"tags_to_id"`
}

func NewIndex() Index {
	return Index{
		IdToNotes: map[int]string{},
		TagsToId:  map[string][]int{},
	}
}

func UnmarshalNote(data []byte) (meta Header, content []byte, err error) {
	data = bytes.TrimLeft(data, "-\n")
	header, content, found := bytes.Cut(data, []byte("---"))
	if !found {
		err = errors.New("could not find header")
		return
	}
	err = yaml.Unmarshal(header, &meta)
	return meta, bytes.TrimSpace(content), err
}
func MarshalNote(meta Header, content []byte) ([]byte, error) {
	header, err := yaml.Marshal(&meta)
	if err != nil {
		return nil, err
	}
	return slicez.Concat([]byte("---\n"), header, []byte("---\n"), content, []byte("\n")), nil
}
