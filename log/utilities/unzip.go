package utilities

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/log/parsers"
)

// Unzip will unzip the file specified by src into the file specified by dest.
func Unzip(src string, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.File {
		filename := f.Name

		// skip file if no parsers run against it
		if !isFileUsed(filename) {
			continue
		}

		fpath, newfpath, err := getFilePath(dest, filename)
		if err != nil {
			return err
		}

		// skip any directories
		if f.FileInfo().IsDir() {
			continue
		}

		// make directory file stored in
		if err = copyFileFromZip(fpath, f); err != nil {
			return err
		}

		err = os.Rename(fpath, newfpath)
		if err != nil {
			return err
		}
	}

	return nil
}

// isFileUsed returns true if the file is needed to run parser functions against.
func isFileUsed(filename string) bool {
	for _, file := range parsers.ParserFunctions {
		// only get log files that parsers run against
		if filepath.Base(filename) == file.Name+".log" || filepath.Base(filename) == "ns_server."+file.Name+".log" {
			return true
		}
	}

	return false
}

// getFilePath creates the filepaths and checks the filepath is legal.
func getFilePath(dest string, filename string) (string, string, error) {
	// Store filename/path for returning and using later on
	fpath := filepath.Join(dest, filename)
	newfpath := filepath.Join(dest, filepath.Base(filename))
	if strings.HasPrefix(filepath.Base(filename), "ns_server") {
		newfpath = filepath.Join(dest, filepath.Base(filename)[10:])
	}

	// Check valid path
	if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("%s: illegal file path", fpath)
	}

	return fpath, newfpath, nil
}

// copyFileFromZip creates a new file in dest and copies the contents of the logfile into it.
func copyFileFromZip(fpath string, f *zip.File) error {
	// make directory file stored in
	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return err
	}

	outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	logFile, err := f.Open()
	if err != nil {
		return err
	}
	defer logFile.Close()

	_, err = io.Copy(outFile, logFile)
	if err != nil {
		return err
	}

	return nil
}
