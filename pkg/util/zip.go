package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
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

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		// Skip the root directory
		if relPath == "." {
			return nil
		}

		if info.IsDir() {
			_, err = zipWriter.Create(relPath + "/")
			return err
		}

		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		// it may be tempting to defer file.Close(), but that might leave open
		// many file descriptors in a large directory. Instead, we'll close it
		// after io.Copy()
		// defer file.Close()

		_, err = io.Copy(zipFileWriter, file)
		if closeErr := file.Close(); closeErr != nil {
			return closeErr
		}
		return err
	})
}
