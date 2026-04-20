package util

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CreateTarLayer creates a deterministic tar archive from a directory
func CreateTarLayer(sourceDir string, writer io.Writer) error {
	tw := tar.NewWriter(writer)
	defer tw.Close()
	
	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		files = append(files, path)
		return nil
	})
	if err != nil { return err }
	
	sort.Strings(files)
	
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil { return err }
		
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil { return err }
		if relPath == "." { continue }
		
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			link, _ = os.Readlink(path)
		}
		
		header, err := tar.FileInfoHeader(info, link)
		if err != nil { return err }
		
		header.Name = filepath.ToSlash(relPath)
		header.ModTime = time.Time{}
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}
		
		if err := tw.WriteHeader(header); err != nil { return err }
		
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil { return err }
			io.Copy(tw, f)
			f.Close()
		}
	}
	return nil
}

// ExtractTar extracts a tar archive to a destination directory
func ExtractTar(reader io.Reader, destDir string) error {
	tr := tar.NewReader(reader)
	
	for {
		header, err := tr.Next()
		if err == io.EOF { break }
		if err != nil { return err }
		
		target := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}
		
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil { return err }
			io.Copy(f, tr)
			f.Close()
			// Force apply execution permissions in case of strict host umask!
			os.Chmod(target, os.FileMode(header.Mode))
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Remove(target) // Clear it if it already exists
			os.Symlink(header.Linkname, target)
		case tar.TypeLink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Remove(target)
			os.Link(filepath.Join(destDir, header.Linkname), target)
		}
	}
	
	return nil
}
