package file

import (
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Walk(paths []string, recursively bool, walkFunc func(filename string, r io.Reader)) {

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			logrus.Warnf("no such file or directory %q: %v", path, err)
			continue
		}
		// handle single file file:
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				logrus.Warnf("unable to open file %q: %v", file.Name(), err)
				continue
			}
			walkFunc(info.Name(), file)
			err = file.Close()
			if err != nil {
				logrus.Warnf("unable to close file %q: %v", file.Name(), err)
			}
			continue
		}
		// handle directory non-recursively:
		if !recursively {
			dir, err := os.Open(path)
			if err != nil {
				logrus.Warnf("unable to open directory %q: %v", dir.Name(), err)
				continue
			}
			files, err := dir.ReadDir(0)
			if err != nil {
				logrus.Warnf("unable to read directory %q: %v", dir.Name(), err)
				continue
			}
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				file, err := os.Open(filepath.Join(path, f.Name()))
				if err != nil {
					logrus.Warnf("unable to open file %q: %v", file.Name(), err)
					continue
				}
				walkFunc(f.Name(), file)
				err = file.Close()
				if err != nil {
					logrus.Warnf("unable to close file %q: %v", file.Name(), err)
				}
				continue
			}
			continue
		}
		// handle directory recursively:
		err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			walkFunc(d.Name(), file)
			err = file.Close()
			if err != nil {
				logrus.Warnf("unable to close file %q: %v", file.Name(), err)
			}
			return nil
		})
		if err != nil {
			logrus.Warnf("unable to open %q: %v", info.Name(), err)
			continue
		}
	}
}
