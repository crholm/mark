package ts

import (
	"bytes"
	"github.com/modfin/henry/slicez"
	"regexp"
	"strings"
)

func GetTagsFromNote(content []byte) []string {
	r := regexp.MustCompile("#[0-9a-zA-Z0-9À-ÖØ-öø-ÿĀ-ƿ_-]+")
	tags := r.FindAll(content, -1)
	return slicez.Map(tags, func(a []byte) string {
		return string(bytes.TrimLeft(a, "#"))
	})
}

func TokenizeText(text string) []string {
	var numbers = regexp.MustCompile("^[0-9]*$")
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return slicez.Filter(slicez.Map(strings.Split(text, " "), func(word string) string {
		word = strings.Trim(strings.TrimSpace(word), ",.-/:;'\"!?")
		word = strings.ToLower(word)
		if numbers.MatchString(word) {
			return ""
		}
		return word
	}), func(s string) bool { return len(s) > 0 })
}
