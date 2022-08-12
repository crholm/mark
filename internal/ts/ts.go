package ts

import (
	"bytes"
	"github.com/modfin/henry/slicez"
	"regexp"
	"strings"
)

func GetTagsFromNote(content []byte) []string {
	r := regexp.MustCompile("#[0-9a-zA-ZÀ-ÖØ-öø-ÿĀ-ƿ_-]+")
	tags := r.FindAll(content, -1)
	return slicez.Map(tags, func(a []byte) string {
		return string(bytes.TrimLeft(a, "#"))
	})
}

func TokenizeText(text string) []string {
	var splitter = regexp.MustCompile("[^a-zA-ZÀ-ÖØ-öø-ÿĀ-ƿ]")
	return slicez.Filter(slicez.Map(splitter.Split(text, -1), func(word string) string {
		return strings.ToLower(strings.TrimSpace(word))
	}), func(s string) bool { return len(s) > 0 })
}
