// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package utilities

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/parsers"

	"github.com/couchbase/tools-common/fsutil"
)

// GetLogsFromCbcollect will unzip the file specified by src into the file specified by dest if a zip file is given or
//
//	move and rename the logs from src to dest if a folder is given.
func GetLogsFromCbcollect(src string, dest string) error {
	if strings.HasSuffix(src, ".zip") {
		reader, err := zip.OpenReader(src)
		if err != nil {
			return err
		}
		defer reader.Close()

		for _, f := range reader.File {
			err = moveFileToDest(f.Name, dest, f, "")
			if err != nil {
				return err
			}
		}

		return nil
	}

	fileInfo, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range fileInfo {
		err = moveFileToDest(file.Name(), dest, nil, filepath.Join(src, file.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

func moveFileToDest(fName string, dest string, zipF *zip.File, osF string) error {
	filename := fName

	// skip file if no parsers run against it
	if !isFileUsed(filename) {
		return nil
	}

	fpath, newfpath, err := getFilePath(dest, filename)
	if err != nil {
		return err
	}

	// skip any directories
	if zipF != nil {
		if zipF.FileInfo().IsDir() {
			return nil
		}
	}

	// make directory file stored in
	if err = copyFileFromZip(fpath, zipF, osF); err != nil {
		return err
	}

	return os.Rename(fpath, newfpath)
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
func copyFileFromZip(path string, zipF *zip.File, osF string) error {
	var (
		reader io.ReadCloser
		err    error
	)

	if zipF != nil {
		reader, err = zipF.Open()
	} else {
		reader, err = os.Open(osF)
	}
	if err != nil {
		return err
	}

	defer reader.Close()

	// make directory file stored in
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	return fsutil.WriteToFile(path, reader, os.ModePerm)
}
