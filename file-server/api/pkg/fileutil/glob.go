package fileutil

import (
	"io/fs"
	"log"
	"path/filepath"
)

// dirをフルパスで渡すとフルパスで返す
func GlobFiles(dir string, pattern string) ([]string, error) {
	matches := []string{}
	if err := filepath.Walk(dir, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		matched, err := filepath.Match(pattern, fileInfo.Name())
		if err != nil {
			return err
		}
		if matched {
			_, err := filepath.Abs(path)
			if err != nil {
				log.Println(err)
			} else {
				matches = append(matches, path)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return matches, nil
}
