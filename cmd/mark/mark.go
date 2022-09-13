package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/crholm/mark"
	"github.com/crholm/mark/internal/fss"
	"github.com/crholm/mark/internal/printer"
	"github.com/crholm/mark/internal/ts"
	"github.com/crholm/mark/internal/tsar"
	"github.com/mattn/go-shellwords"
	"github.com/modfin/henry/compare"
	"github.com/modfin/henry/mapz"
	"github.com/modfin/henry/slicez"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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
						cmd := exec.Command("/proc/self/exe", args...)
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
				Name:    "pager",
				Aliases: []string{"page", "p"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "pick",
						Aliases: []string{"p"},
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

					if c.Bool("pick") {
						file, err := pickFile(files)
						if err != nil {
							return err
						}
						files = []string{file}
					}

					return page(files, printer.Of(c.String("format")))
				},
			},
			{
				Name:    "cat",
				Aliases: []string{"c"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "pick",
						Aliases: []string{"p"},
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

					if c.Bool("pick") {
						file, err := pickFile(files)
						if err != nil {
							return err
						}
						files = []string{file}
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
				Name: "ll",
				Action: func(c *cli.Context) error {
					prefix := c.Args().First()

					files, err := ls(prefix)
					if err != nil {
						return err
					}
					slicez.Each(slicez.Sort(files), func(f string) {
						name, _ := ll(f)
						fmt.Println(name)
					})
					return err
				},
			},
			{
				Name: "rm",
				Action: func(c *cli.Context) error {
					filename := c.Args().First()

					file, err := fss.GetFilenameToPath(filename)
					if err != nil {
						return err
					}
					fmt.Println("rm", file)
					return os.Remove(file)
				},
			},
			{
				Name:    "edit",
				Aliases: []string{"e"},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "raw",
					},
				},
				Action: editNote,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format"},
			&cli.StringFlag{Name: "mode"},
		},
		Action: newNote,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func doEdit(file string) error {
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

	file, err := pickFile(files)
	if err != nil {
		return err
	}

	defer updateIndex(file)
	if c.Bool("raw") {
		return editFile(file)
	}
	return doEdit(file)
}

func ll(f string) (string, error) {

	file, err := os.Open(f)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	header := ""
	_, _, err = buf.ReadLine() // discard
	if err != nil {
		return "", err
	}
	line := ""
	for line != "---" {
		b, _, err := buf.ReadLine() // discard
		if err != nil {
			return "", err
		}
		line = string(b)
		header += fmt.Sprintln(line)
	}
	var meta mark.Header
	err = yaml.Unmarshal([]byte(header), &meta)
	if err != nil {
		return "", err
	}

	title := strings.TrimSpace(meta.Title)
	tags := meta.Tags
	var parts []string
	if len(title) > 0 {
		parts = append(parts, title)
	}
	if len(tags) > 0 {
		parts = append(parts, fmt.Sprint(tags))
	}
	name := filepath.Base(f)
	return fmt.Sprintf("%s %s %s", name, strings.Repeat(" ", 35-len(name)), strings.Join(parts, " ")), nil
}

func pickFile(files []string) (string, error) {

	if len(files) == 1 {
		return files[0], nil
	}
	picker := os.Getenv("MARK_PICKER")
	if len(picker) > 0 {
		parts, err := shellwords.Parse(picker)
		if err != nil {
			return "", err
		}
		cmd := exec.Command(parts[0], parts[1:]...)

		reader, writer := io.Pipe()
		cmd.Stdin = reader

		go func() {
			mode := os.Getenv("MARK_PICKER_MODE")
			if mode != "grep" {
				mode = "file"
			}

			switch mode {
			case "grep":
				_, _ = io.Copy(writer, cat(files, printer.AnnotatedPrinter))
			default:
				for _, f := range files {
					s, _ := ll(f)
					_, _ = writer.Write([]byte(s + "\n"))
				}
			}
			_ = writer.Close()
		}()

		buf := bytes.NewBuffer(nil)
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			return "", err
		}
		b, err := ioutil.ReadAll(buf)
		if err != nil {
			return "", err
		}
		fmt.Println("FILE", string(b))
		file, _, _ := strings.Cut(strings.TrimSpace(string(b)), " ")
		file = strings.TrimRight(file, ":1234567890")
		if len(file) == 0 {
			return "", errors.New("no selected file")
		}

		f, _ := slicez.Find(files, func(e string) bool {
			return strings.HasSuffix(e, file)
		})

		return f, nil
	}

	files = slicez.Take(files, 10)

	for i, f := range files {
		name, err := ll(f)
		if err != nil {
			return "", err
		}
		fmt.Println(i+1, "-", name)
	}
	fmt.Print("? ")
	var i int
	_, err := fmt.Scanf("%d", &i)
	if err != nil {
		return "", err
	}
	return slicez.Nth(files, i-1), nil
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
	filename, err := fss.SaveNote(meta, note)
	if err != nil {
		return err
	}

	if c.Args().Len() == 0 {
		doEdit(filename)
	}

	return updateIndex(filename)
}

func ls(prefix string) ([]string, error) {

	if prefix == "-" {
		reader := bufio.NewReader(os.Stdin)
		line, _, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		prefix = strings.TrimSpace(string(line))
	}

	var filename = regexp.MustCompile("^([0-9]{3,4})")
	if len(prefix) == 0 || filename.MatchString(prefix) {

		year := "*"
		month := "*"

		if len(prefix) > 3 {
			year = prefix[0:4]
		}
		if len(prefix) > 6 {
			month = prefix[5:7]
		}

		glob := fmt.Sprintf("%s/%s/%s/%s*", fss.GetLibPath(), year, month, prefix)
		files, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		if len(files) > 0 {
			return slicez.Reverse(slicez.Sort(files)), nil
		}
	}

	var specificTag = prefix[0] == ':' || prefix[0] == '#'
	if specificTag {
		prefix = prefix[1:]
	}

	var data []byte
	var err error

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
		f, err := fss.GetFilenameToPath(index.IdToName[a])
		if err != nil {
			return ""
		}
		if _, err := os.Stat(f); errors.Is(err, os.ErrNotExist) {
			return ""
		}
		return f
	})

	var tsFiles []string
	if !specificTag {
		data, err = ioutil.ReadFile(filepath.Join(fss.GetStoragePath(), "index.tsar"))
		if err != nil {
			return nil, err
		}
		tsIndex, err := tsar.UnmarshalIndex(data)
		if err != nil {
			return nil, err
		}
		entries, err := tsIndex.Find(strings.ToLower(prefix), tsar.MatchPrefix)
		if err != nil {
			return nil, err
		}

		tsFiles = slicez.Flatten(slicez.Map(entries, func(entry *tsar.Entry) []string {
			return slicez.Map(entry.Pointers, func(a uint32) string {
				f, err := fss.GetFilenameToPath(index.IdToName[int(a)])
				if err != nil {
					return ""
				}
				if _, err := os.Stat(f); errors.Is(err, os.ErrNotExist) {
					return ""
				}
				return f
			})
		}))
	}

	return slicez.Uniq(slicez.Filter(append(tagFiles, tsFiles...), func(s string) bool {
		return len(s) > 0
	})), nil
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
	err := cmd.Run()
	if err != nil {
		fmt.Println("The err", err)
		return err
	}
	return nil
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

			data = printer(header, content, file)
			_, err = io.Copy(w, bytes.NewReader(data))

			if err != nil {
				panic(err)
			}
		}
	}()

	return r
}

func updateIndex(file string) error {

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	header, content, err := mark.UnmarshalNote(data)
	if err != nil {
		return err
	}

	// Updating json mapping index

	jsonIndexName := filepath.Join(fss.GetStoragePath(), "index.json")
	var jsonindexdata = []byte("{}")

	if _, err := os.Stat(jsonIndexName); err == nil {
		jsonindexdata, err = ioutil.ReadFile(jsonIndexName)
		if err != nil {
			return err
		}
	}

	var jsonindex = mark.NewIndex()
	err = json.Unmarshal(jsonindexdata, &jsonindex)
	if err != nil {
		return err
	}

	name := filepath.Base(file)
	maxId := slicez.Max[int](mapz.Keys(jsonindex.IdToName)...)
	nameToId := mapz.Remap(jsonindex.IdToName, func(k int, v string) (string, int) {
		return v, k
	})
	id, found := nameToId[name]
	if !found {
		id = maxId + 1
		jsonindex.IdToName[id] = name
	}

	// Remove old tags from index
	for _, tag := range jsonindex.IdToTags[id] {
		jsonindex.TagsToId[tag] = slicez.Reject(jsonindex.TagsToId[tag], compare.EqualOf(id))
	}

	// Adds current tags
	jsonindex.IdToTags[id] = append([]string{}, header.Tags...)
	for _, tag := range jsonindex.IdToTags[id] {
		tags := jsonindex.TagsToId[tag]
		jsonindex.TagsToId[tag] = slicez.Uniq(append(tags, id))
	}

	jsonindexdata, err = json.Marshal(jsonindex)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(jsonIndexName, jsonindexdata, 0644)
	if err != nil {
		return err
	}

	// Updating tsindexdata could be slow eventually?
	tsarIndexName := filepath.Join(fss.GetStoragePath(), "index.tsar")
	var tsindex = &tsar.Index{}
	var tsindexdata []byte
	if _, err := os.Stat(tsarIndexName); err == nil {
		tsindexdata, err = ioutil.ReadFile(tsarIndexName)
		if err != nil {
			return err
		}
		tsindex, err = tsar.UnmarshalIndex(tsindexdata)
		if err != nil {
			return err
		}
	}

	wordlist := tsindex.EntryList()
	for _, word := range ts.TokenizeText(string(content)) {
		err = wordlist.Append(word, uint32(id))
		if err != nil {
			return err
		}
	}
	tsindex = wordlist.ToIndex()
	tsindexdata = tsar.MarshalIndex(tsindex)
	err = ioutil.WriteFile(tsarIndexName, tsindexdata, 0644)
	if err != nil {
		return err
	}

	return nil
}

func reindex(c *cli.Context) error {

	index := mark.NewIndex()
	wordlist := tsar.NewEntryList()

	files, err := ls("")
	if err != nil {
		return err
	}

	for id, f := range slicez.Sort(files) {
		index.IdToName[id] = filepath.Base(f)
		data, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		header, content, err := mark.UnmarshalNote(data)
		if err != nil {
			return err
		}
		index.IdToTags[id] = header.Tags
		for _, tag := range header.Tags {
			tags := index.TagsToId[tag]
			index.TagsToId[tag] = append(tags, id)
		}
		for _, word := range ts.TokenizeText(string(content)) {
			err = wordlist.Append(word, uint32(id))
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
