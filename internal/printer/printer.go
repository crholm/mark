package printer

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/glamour"
	"github.com/crholm/mark"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Printer = func(header mark.Header, raw []byte, filename string) []byte

func Of(printer string) Printer {
	switch printer {
	case "raw":
		return RawPrinter
	case "plain":
		return PlainPrinter
	case "annotated":
		return AnnotatedPrinter
	default:
		return FormattedPrinter
	}
}

func RawPrinter(header mark.Header, raw []byte, _ string) []byte {
	data, err := mark.MarshalNote(header, raw)
	if err != nil {
		panic(err)
	}
	return append(append([]byte(nil), data...), []byte("\n")...)
}

func PlainPrinter(header mark.Header, raw []byte, _ string) []byte {
	title := fmt.Sprintf("--- %s --- %s\n", header.CreatedAt.In(time.Local).Format("Monday Jan 02 2006 - 15:04:05"), header.Title)

	content := append([]byte(" "), bytes.ReplaceAll(raw, []byte("\n"), []byte("\n "))...)
	footer := "\n\n"
	return append(append([]byte(title), content...), []byte(footer)...)

}

func AnnotatedPrinter(header mark.Header, raw []byte, file string) []byte {
	anno := []byte(filepath.Base(file) + ":")
	content := RawPrinter(header, raw, "")

	var res []byte
	for i, line := range bytes.Split(content, []byte("\n")) {
		l := append(anno, []byte(strconv.Itoa(i+1)+": ")...)
		l = append(l, line...)
		l = append(l, '\n')
		res = append(res, l...)
	}
	return res
}

func FormattedPrinter(header mark.Header, raw []byte, _ string) []byte {
	width := 110

	title := ""
	if len(header.Title) > 0 {
		title = fmt.Sprint(" ", header.Title, " ")
	}

	title = fmt.Sprintf("┌─%s────",
		title,
	)
	timestamp := header.CreatedAt.In(time.Local).Format("Monday Jan 02 2006 - 15:04:05")
	title = fmt.Sprintf("%s%s %s\n", title, strings.Repeat("─", width-len([]rune(title))-len(timestamp)-2), timestamp)

	render, err := glamour.NewTermRenderer(glamour.WithEnvironmentConfig(), glamour.WithWordWrap(width))
	if err != nil {
		panic(err)
	}
	out, err := render.RenderBytes(raw)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var content string
	for i, s := range lines {
		s := strings.TrimSpace(s)
		content += "│" + s
		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content += "\n"
		}
	}
	footer := "\n└" + strings.Repeat("─", len([]rune(title))-2) + "\n\n"
	return append(append([]byte(title), []byte(content)...), []byte(footer)...)

}
