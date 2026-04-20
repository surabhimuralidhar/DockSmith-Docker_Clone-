package util

import (
	"io"
	"os"
	"path/filepath"
)

// MatchGlob matches files using glob patterns (* and **)
func MatchGlob(baseDir, pattern string) ([]string, error) {
	var matches []string

	if pattern == "." {
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				matches = append(matches, path)
			}
			return nil
		})
		return matches, err
	}

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			return err
		}

		if matched && info.Mode().IsRegular() {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

// CopyFile copies a file from src to dest
func CopyFile(srcPath, destPath string) error {
	err := os.MkdirAll(filepath.Dir(destPath), 0755)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
