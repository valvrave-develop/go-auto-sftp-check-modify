package resource

import (
	"bytes"
	"dir"
	"fmt"
	"os"
	"path/filepath"
	"project"
	"strings"
)

type Resource interface {
	Get(string) string
}

type DirTreeResource struct {
	Project *project.Project
}

func NewDirTreeResource(project *project.Project) Resource{
	return &DirTreeResource{
		Project:project,
	}
}

func (d *DirTreeResource) Get(path string) string {
	buffer := new(bytes.Buffer)
	format := func(n int) error {
		if n == 0 {
			return nil
		}
		for i:=0; i<n; i++ {
			err := buffer.WriteByte('-')
			if err != nil {
				return err
			}
		}
		err := buffer.WriteByte('>')
		if err != nil {
			return err
		}
		return nil
	}
	osPath := strings.Join(strings.Split(path, "/"), fmt.Sprintf("%c", os.PathSeparator))
	basePath := filepath.Dir(d.Project.Dirs.Dir.Name)
	obsPath := fmt.Sprintf("%s%c%s", basePath, os.PathSeparator, osPath)
	if _, ok := d.Project.Dirs.Index[obsPath]; !ok {
		return "Not Found"
	}
	err := printDirTree(d.Project.Dirs.Index[obsPath], 0, format, buffer)
	if err != nil {
		return "error"
	}
	return buffer.String()
}

func printDirTree(dir *dir.Dir, n int, format func(n int) error, w *bytes.Buffer) error {
	err := format(2 * n)
	if err != nil {
		return err
	}
	_, err = w.WriteString(fmt.Sprintf("%s:%d\n", filepath.Base(dir.Name), dir.Status))
	if err != nil {
		return err
	}
	for _, file := range dir.Files{
		err := format(2 * (n+1))
		if err != nil {
			return err
		}
		_, err = w.WriteString(fmt.Sprintf("%s:%d\n", filepath.Base(file.Name), file.Status))
		if err != nil {
			return err
		}
	}
	for _, child := range dir.ChildDirs{
		err = printDirTree(child, n+1, format, w)
		if err != nil {
			return err
		}
	}
	return nil
}