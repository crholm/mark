package mark

import (
	"bytes"
	"errors"
	"github.com/modfin/henry/slicez"
	"gopkg.in/yaml.v3"
	"time"
)

type Header struct {
	Title     string    `yaml:"title"`
	Alias     string    `yaml:"alias"`
	Tags      []string  `yaml:"tags"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

type Index struct {
	IdToName map[int]string   `json:"id_to_name"`
	TagsToId map[string][]int `json:"tags_to_id"`
	IdToTags map[int][]string `json:"id_to_tags"`
}

func NewIndex() Index {
	return Index{
		IdToName: map[int]string{},
		TagsToId: map[string][]int{},
		IdToTags: map[int][]string{},
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
