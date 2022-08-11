package mark

import "time"

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
