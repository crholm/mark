package printer

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/glamour"
	"mark"
	"strings"
	"time"
)

type Printer = func(header mark.Header, raw []byte) []byte

func Of(printer string) Printer {
	switch printer {
	case "raw":
		return RawPrinter
	case "plain":
		return PlainPrinter
	default:
		return FormattedPrinter
	}
}

func RawPrinter(header mark.Header, raw []byte) []byte {
	data, err := mark.MarshalNote(header, raw)
	if err != nil {
		panic(err)
	}
	return append(append([]byte(nil), data...), []byte("\n")...)
}

func PlainPrinter(header mark.Header, raw []byte) []byte {
	title := fmt.Sprintf("--- %s --- %s\n", header.CreatedAt.In(time.Local).Format("Monday Jan 02 2006 - 15:04:05"), header.Title)

	content := append([]byte(" "), bytes.ReplaceAll(raw, []byte("\n"), []byte("\n "))...)
	footer := "\n\n"
	return append(append([]byte(title), content...), []byte(footer)...)

}

func FormattedPrinter(header mark.Header, raw []byte) []byte {
	width := 110

	title := ""
	if len(header.Title) > 0 {
		title = fmt.Sprint(" ", header.Title, " ")
	}

	title = fmt.Sprintf("┌─%s──── %s",
		title,
		header.CreatedAt.In(time.Local).Format("Monday Jan 02 2006 - 15:04:05"),
	)

	title = fmt.Sprintf("%s%s\n", title, strings.Repeat("─", width-len([]rune(title))))

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
