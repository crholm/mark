package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/modfin/henry/slicez"
	"github.com/urfave/cli/v2"
	"io"
	"io/ioutil"
	"log"
	"mark"
	"mark/internal/fss"
	"mark/internal/printer"
	"mark/internal/ts"
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
				Name:   "reindex",
				Action: reindex,
			},
			{
				Name: "git",
				Action: func(c *cli.Context) error {

					exe := func(args []string) error {
						cmd := exec.Command("git", args...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Dir = fss.GetStoragePath()
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

					return page(files, printer.Of(c.String("format")))
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

					_, err = io.Copy(os.Stdout, cat(files, printer.Of(c.String("format"))))
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
				Action: editNote,
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

func editNote(c *cli.Context) error {
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
	meta, content, err := mark.UnmarshalNote(data)
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
	meta.Tags = ts.GetTagsFromNote(content)
	data, err = mark.MarshalNote(meta, content)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, data, 0644)
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

	meta.Tags = ts.GetTagsFromNote(contentBytes)
	note, err := mark.MarshalNote(meta, contentBytes)
	if err != nil {
		return err
	}
	err = fss.SaveNote(meta, note)
	if err != nil {
		return err
	}
	return nil
}

func ls(prefix string) ([]string, error) {
	var date = regexp.MustCompile("^[0-9-:]*$")

	if len(prefix) == 0 || date.MatchString(prefix) {
		files, err := filepath.Glob(fmt.Sprintf("%s/*/*/%s*.md", fss.GetLibPath(), prefix))
		if err != nil {
			return nil, err
		}
		return slicez.Reverse(slicez.Sort(files)), nil
	}

	data, _ := ioutil.ReadFile(filepath.Join(fss.GetStoragePath(), "index.tsar"))
	tsIndex, err := tsar.UnmarshalIndex(data)
	if err != nil {
		return nil, err
	}
	entries, err := tsIndex.Find(prefix, tsar.MatchPrefix)
	if err != nil {
		return nil, err
	}

	index := mark.NewIndex()
	data, err = ioutil.ReadFile(filepath.Join(fss.GetStoragePath(), "index.json"))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &index)
	if err != nil {
		return nil, err
	}

	tagFiles := slicez.Map(index.TagsToId[prefix], func(a int) string {
		f, err := fss.GetFilenameToPath(index.IdToNotes[a])
		if err != nil {
			return ""
		}
		return f
	})

	tsFiles := slicez.Uniq(slicez.Flatten(slicez.Map(entries, func(entry *tsar.Entry) []string {
		return slicez.Map(entry.Pointers, func(a uint32) string {
			f, err := fss.GetFilenameToPath(index.IdToNotes[int(a)])
			if err != nil {
				return ""
			}
			return f
		})
	})))

	return append(tagFiles, tsFiles...), nil
}

func page(files []string, printer printer.Printer) error {

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

func cat(files []string, printer printer.Printer) io.Reader {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		for _, file := range files {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				panic(err)
			}

			header, content, err := mark.UnmarshalNote(data)
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

func reindex(c *cli.Context) error {

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
		header, content, err := mark.UnmarshalNote(data)
		if err != nil {
			return err
		}
		for _, tag := range header.Tags {
			tags := index.TagsToId[tag]
			index.TagsToId[tag] = append(tags, i)
		}
		for _, word := range ts.TokenizeText(string(content)) {
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

	err = ioutil.WriteFile(filepath.Join(fss.GetStoragePath(), "index.json"), jsonindex, 0644)
	if err != nil {
		return err
	}

	tsindex := tsar.MarshalIndex(wordlist.ToIndex())
	err = ioutil.WriteFile(filepath.Join(fss.GetStoragePath(), "index.tsar"), tsindex, 0644)
	if err != nil {
		return err
	}

	return nil
}
