package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/glamour"
	"github.com/modfin/henry/slicez"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"mark"
	"mark/internal/tsar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	app := &cli.App{
		Name:  "mark",
		Usage: "Taken notes by writing things....",
		Commands: []*cli.Command{
			{
				Name: "reindex",
				Action: func(c *cli.Context) error {

					index := mark.Index{
						IdToNotes: map[int]string{},
						TagsToId:  map[string][]int{},
					}
					wordlist := tsar.NewEntryList()

					files, err := ls("")
					if err != nil {
						return err
					}

					for i, f := range slicez.Sort(files) {
						index.IdToNotes[i] = filepath.Base(f)
						data, err := ioutil.ReadFile(f)
						if err != nil {
							return err
						}
						header, content, err := unmarshalNote(data)
						if err != nil {
							return err
						}
						for _, tag := range header.Tags {
							tags := index.TagsToId[tag]
							index.TagsToId[tag] = append(tags, i)
						}
						for _, word := range tokenizeText(string(content)) {
							err = wordlist.Append(word, uint32(i))
							if err != nil {
								return err
							}
						}
					}

					jsonindex, err := json.Marshal(index)
					if err != nil {
						return err
					}

					err = ioutil.WriteFile(filepath.Join(getStoragePath(), "index.json"), jsonindex, 0644)
					if err != nil {
						return err
					}

					tsindex := tsar.MarshalIndex(wordlist.ToIndex())
					err = ioutil.WriteFile(filepath.Join(getStoragePath(), "index.tsar"), tsindex, 0644)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name: "git",
				Action: func(c *cli.Context) error {

					exe := func(args []string) error {
						cmd := exec.Command("git", args...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Dir = getStoragePath()
						return cmd.Run()
					}

					return exe(c.Args().Slice())
				},
			},
			{
				Name: "sync",
				Action: func(c *cli.Context) error {

					exe := func(args []string) error {
						cmd := exec.Command("/proc/self", args...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						return cmd.Run()
					}

					err := exe([]string{"git", "add", "."})
					if err != nil {
						fmt.Println("mark: are you sure you have git installed and the repo initialized? see `mark sync`", err)
						return nil
					}
					exe([]string{"git", "commit", "-m", "sync commit"})
					exe([]string{"git", "pull"})
					exe([]string{"git", "push"})
					return nil
				},
			},
			{
				Name: "pager",
				Action: func(c *cli.Context) error {
					prefix := c.Args().First()

					files, err := ls(prefix)
					if err != nil {
						return err
					}

					if len(files) == 0 {
						fmt.Println("no entries")
						return nil
					}

					return page(files, getPrinter(c.String("format")))
				},
			},
			{
				Name: "cat",
				Action: func(c *cli.Context) error {
					prefix := c.Args().First()

					files, err := ls(prefix)
					if err != nil {
						return err
					}

					if len(files) == 0 {
						fmt.Println("no entries")
						return nil
					}

					_, err = io.Copy(os.Stdout, cat(files, getPrinter(c.String("format"))))
					return err
				},
			},
			{
				Name: "ls",
				Action: func(c *cli.Context) error {
					prefix := c.Args().First()

					files, err := ls(prefix)
					if err != nil {
						return err
					}
					slicez.Each(slicez.Sort(files), func(f string) {
						fmt.Println(filepath.Base(f))
					})
					return err
				},
			},
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "offset",
						Aliases: []string{"o"},
					},
					&cli.BoolFlag{
						Name: "raw",
					},
				},
				Action: func(c *cli.Context) error {
					prefix := c.Args().First()

					files, err := ls(prefix)
					if err != nil {
						return err
					}
					if len(files) == 0 {
						fmt.Println("no entries")
						return nil
					}
					file := slicez.Nth(files, c.Int("offset"))

					if c.Bool("raw") {
						return editFile(file)
					}

					data, err := ioutil.ReadFile(file)
					if err != nil {
						return err
					}
					meta, content, err := unmarshalNote(data)
					if err != nil {
						return err
					}

					f, err := os.CreateTemp("", "mark.*.md")
					if err != nil {
						return err
					}
					defer os.Remove(f.Name())
					_, err = f.Write(content)
					if err != nil {
						return err
					}
					err = f.Close()
					err = editFile(f.Name())
					if err != nil {
						return err
					}
					content, err = ioutil.ReadFile(f.Name())
					if err != nil {
						return err
					}

					meta.UpdatedAt = time.Now()
					meta.Tags = getTagsFromNote(content)
					data, err = marshalNote(meta, content)
					if err != nil {
						return err
					}
					return ioutil.WriteFile(file, data, 0644)
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format"},
		},
		Action: newNote,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newNote(c *cli.Context) error {
	meta := mark.Header{
		Title:     c.String("t"),
		Tags:      nil,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	title, content, found := slicez.Cut(c.Args().Slice(), "--")
	if !found {
		content = title
	}

	if found {
		meta.Title = strings.Join(title, " ")
	}

	contentBytes := []byte(strings.Join(content, " "))

	meta.Tags = getTagsFromNote(contentBytes)
	note, err := marshalNote(meta, contentBytes)
	if err != nil {
		return err
	}
	err = saveNote(meta, note)
	if err != nil {
		return err
	}
	return nil
}

func ls(prefix string) ([]string, error) {
	files, err := filepath.Glob(fmt.Sprintf("%s/*/*/%s*.md", getLibPath(), prefix))
	if err != nil {
		return nil, err
	}

	if len(prefix) > 0 {
		data, _ := ioutil.ReadFile(filepath.Join(getStoragePath(), "index.tsar"))
		i, err := tsar.UnmarshalIndex(data)
		if err != nil {
			return nil, err
		}

		e, err := i.Find(prefix, tsar.MatchPrefix)
		if err != nil {
			return nil, err
		}
		for _, ee := range e {
			fmt.Println("look for file", ee)
		}
	}
	return slicez.Reverse(slicez.Sort(files)), nil
}

func page(files []string, printer printer) error {

	pager := os.Getenv("PAGER")
	if len(pager) == 0 {
		pager = "less -r"
	}

	pa := strings.Split(pager, " ")
	cmd := exec.Command(pa[0], pa[1:]...)
	b, _ := ioutil.ReadAll(cat(files, printer))
	cmd.Stdin = bytes.NewReader(b) // for some reason having a reader with all the content from the beginning works with less, but not a regular reader.
	//cmd.Stdin = cat(files)
	//cmd.Stdin = os.Stdin
	//cmd.Stdin = bufio.NewReader(cat(files))
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func editFile(file string) error {

	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "nano"
	}

	pa := strings.Split(editor, " ")
	pa = append(pa, file)
	cmd := exec.Command(pa[0], pa[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func cat(files []string, printer printer) io.Reader {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		for _, file := range files {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				panic(err)
			}

			header, content, err := unmarshalNote(data)
			if err != nil {
				panic(err)
			}

			data = printer(header, content)
			_, err = io.Copy(w, bytes.NewReader(data))

			if err != nil {
				panic(err)
			}
		}
	}()

	return r
}

func getPrinter(printer string) printer {
	switch printer {
	case "raw":
		return rawPrinter
	case "plain":
		return plainPrinter
	default:
		return formattedPrinter
	}
}

type printer = func(header mark.Header, raw []byte) []byte

func rawPrinter(header mark.Header, raw []byte) []byte {
	data, err := marshalNote(header, raw)
	if err != nil {
		panic(err)
	}
	return append(append([]byte(nil), data...), []byte("\n")...)
}

func plainPrinter(header mark.Header, raw []byte) []byte {
	title := fmt.Sprintf("--- %s --- %s\n", header.CreatedAt.In(time.Local).Format("Monday Jan 02 2006 - 15:04:05"), header.Title)

	content := append([]byte(" "), bytes.ReplaceAll(raw, []byte("\n"), []byte("\n "))...)
	footer := "\n\n"
	return append(append([]byte(title), content...), []byte(footer)...)

}

func formattedPrinter(header mark.Header, raw []byte) []byte {
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

func getFilename(meta mark.Header) string {
	return fmt.Sprintf("%s.md", meta.CreatedAt.In(time.UTC).Format("2006-01-02T15:04:05Z0700_Monday"))
}
func getLibPath() string {
	return filepath.Join(getStoragePath(), "lib")
}
func getPath(meta mark.Header) string {
	return filepath.Join(getLibPath(), meta.CreatedAt.Format("2006"), meta.CreatedAt.Format("01"))
}
func getFullPath(meta mark.Header) string {
	return filepath.Join(getPath(meta), getFilename(meta))
}

func saveNote(meta mark.Header, content []byte) error {
	_ = os.MkdirAll(getPath(meta), 0755)
	err := ioutil.WriteFile(getFullPath(meta), content, 0644)
	return err
}

func unmarshalNote(data []byte) (meta mark.Header, content []byte, err error) {
	data = bytes.TrimLeft(data, "-\n")
	header, content, found := bytes.Cut(data, []byte("---"))
	if !found {
		err = errors.New("could not find header")
		return
	}
	err = yaml.Unmarshal(header, &meta)
	return meta, bytes.TrimSpace(content), err
}

func getTagsFromNote(content []byte) []string {
	r := regexp.MustCompile("#[0-9a-zA-Z0-9À-ÖØ-öø-ÿĀ-ƿ_-]+")
	tags := r.FindAll(content, -1)
	return slicez.Map(tags, func(a []byte) string {
		return string(bytes.TrimLeft(a, "#"))
	})
}

func marshalNote(meta mark.Header, content []byte) ([]byte, error) {
	header, err := yaml.Marshal(&meta)
	if err != nil {
		return nil, err
	}
	return slicez.Concat([]byte("---\n"), header, []byte("---\n"), content, []byte("\n")), nil
}

func getStoragePath() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirname, ".mark")
	_ = os.MkdirAll(path, 0755)
	return path
}

func tokenizeText(text string) []string {
	numbers := regexp.MustCompile("^[0-9]*$")
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
