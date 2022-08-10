package main

import (
	"bytes"
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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	app := &cli.App{
		Name:  "mark",
		Usage: "Taken notes by writing things....",
		Commands: []*cli.Command{
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

					return page(files)
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

					_, err = io.Copy(os.Stdout, cat(files))
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

					return edit(file)
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
			},
		},
		Action: func(c *cli.Context) error {
			meta := mark.Meta{
				Title:     c.String("t"),
				Tags:      nil,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			args, content, found := slicez.Cut(c.Args().Slice(), "--")
			if !found {
				content = args
			}
			contentBytes := []byte(strings.Join(content, " "))

			note, err := marshalNote(meta, contentBytes)
			if err != nil {
				log.Fatal(err)
			}
			err = saveNote(meta, note)
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func ls(prefix string) ([]string, error) {
	files, err := filepath.Glob(fmt.Sprintf("%s/*/*/%s*.md", getLibPath(), prefix))
	if err != nil {
		return nil, err
	}
	return slicez.Reverse(slicez.Sort(files)), nil
}

func page(files []string) error {

	pager := os.Getenv("PAGER")
	if len(pager) == 0 {
		pager = "less -r"
	}

	r := cat(files)
	b, _ := ioutil.ReadAll(r)

	pa := strings.Split(pager, " ")
	cmd := exec.Command(pa[0], pa[1:]...)
	cmd.Stdin = bytes.NewReader(b) // for some reason having a reader with all the content from the begining works with less, but not a regular reader.
	//cmd.Stdin = bufio.NewReader(r)
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func edit(file string) error {

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

func cat(files []string) io.Reader {
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

func printer(header mark.Meta, raw []byte) []byte {
	title := fmt.Sprintf("┌── %s ────", header.CreatedAt.In(time.Local).Format("Monday Jan 02 2006 - 15:04:05"))
	if len(header.Title) > 0 {
		title = fmt.Sprint(title, " ", header.Title, " ────")
	}
	title = fmt.Sprintln(title)
	out, err := glamour.RenderBytes(raw, "auto")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var content string
	for i, s := range lines {
		content += "│" + strings.TrimSpace(s)

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content += "\n"
		}
	}
	footer := "\n└" + strings.Repeat("─", len([]rune(title))-2) + "\n\n"
	return append(append([]byte(title), []byte(content)...), []byte(footer)...)

}

func getFilename(meta mark.Meta) string {
	return fmt.Sprintf("%s.md", meta.CreatedAt.In(time.UTC).Format("2006-01-02T15:04:05Z0700_Monday"))
}
func getLibPath() string {
	return filepath.Join(getHomeDir(), "lib")
}
func getPath(meta mark.Meta) string {
	return filepath.Join(getLibPath(), meta.CreatedAt.Format("2006"), meta.CreatedAt.Format("01"))
}
func getFullPath(meta mark.Meta) string {
	return filepath.Join(getPath(meta), getFilename(meta))
}

func saveNote(meta mark.Meta, content []byte) error {
	_ = os.MkdirAll(getPath(meta), 0755)
	err := ioutil.WriteFile(getFullPath(meta), content, 0644)
	return err
}

func unmarshalNote(data []byte) (meta mark.Meta, content []byte, err error) {
	data = bytes.TrimLeft(data, "-\n")
	header, content, found := bytes.Cut(data, []byte("---"))
	if !found {
		err = errors.New("could not find header")
		return
	}
	err = yaml.Unmarshal(header, &meta)
	return meta, bytes.TrimSpace(content), err
}

func marshalNote(meta mark.Meta, content []byte) ([]byte, error) {
	header, err := yaml.Marshal(&meta)
	if err != nil {
		return nil, err
	}
	return slicez.Concat([]byte("---\n"), header, []byte("\n---\n"), content, []byte("\n")), nil
}

func getHomeDir() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirname, ".mark")
	_ = os.MkdirAll(path, 0755)
	return path
}
