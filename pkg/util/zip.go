package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/boyter/gocodewalker"
)

// ZipDirectory compresses the given source directory into the destination file path.
func ZipDirectory(srcDir, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Use gocodewalker to respect .gitignore/.ignore files while collecting paths
	fileQueue := make(chan *gocodewalker.File, 256)
	walker := gocodewalker.NewFileWalker(srcDir, fileQueue)
	// Include hidden files (to match previous behaviour) but still respect .gitignore rules
	walker.IncludeHidden = true

	// Start walking in a separate goroutine so we can process files as they arrive
	go func() {
		_ = walker.Start()
	}()

	// Track directories we've already added to the zip archive so we don't duplicate entries
	dirsAdded := make(map[string]struct{})

	for f := range fileQueue {
		// Compute path in archive using forward slashes
		relPath, err := filepath.Rel(srcDir, f.Location)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Ensure parent directories exist in the archive
		if dir := filepath.Dir(relPath); dir != "." && dir != "" {
			// Walk up the directory tree ensuring each level exists
			segments := strings.Split(dir, "/")
			var current string
			for _, segment := range segments {
				if current == "" {
					current = segment
				} else {
					current = current + "/" + segment
				}
				if _, exists := dirsAdded[current+"/"]; !exists {
					if _, err := zipWriter.Create(current + "/"); err != nil {
						return err
					}
					dirsAdded[current+"/"] = struct{}{}
				}
			}
		}

		// Create the file inside the zip archive
		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(f.Location)
		if err != nil {
			return err
		}
		// Avoid deferring to reduce open FDs on huge trees
		_, err = io.Copy(zipFileWriter, file)
		if closeErr := file.Close(); closeErr != nil {
			return closeErr
		}
		if err != nil {
			return err
		}
	}

	return nil
}
