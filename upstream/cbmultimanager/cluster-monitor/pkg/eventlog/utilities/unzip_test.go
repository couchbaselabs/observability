// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package utilities

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/parsers"

	"github.com/stretchr/testify/require"
)

type unzipTestCase struct {
	name          string
	destPath      string
	srcPath       string
	errorExpected bool
}

func TestUnzip(t *testing.T) {
	currentPath := t.TempDir()

	err := os.MkdirAll(filepath.Join(currentPath, "newSrcFolder"), os.ModePerm)
	require.NoError(t, err)

	testCases := []unzipTestCase{
		{
			name:     "currentDestPathZip",
			destPath: currentPath,
			srcPath:  "zipped_file.zip",
		},
		{
			name:     "newDestPathZip",
			destPath: currentPath + "/newFolder",
			srcPath:  "zipped_file.zip",
		},
		{
			name:     "currentDestPathFolder",
			destPath: currentPath,
			srcPath:  "logs",
		},
		{
			name:     "newDestPathFolder",
			destPath: currentPath + "/newFolder",
			srcPath:  "logs",
		},
		{
			name:     "newSourcePath",
			destPath: currentPath,
			srcPath:  "/newSrcFolder/zipped_file.zip",
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			err := os.RemoveAll(x.srcPath)
			require.NoError(t, err)

			parsers.ParserFunctions = []parsers.Log{
				{
					Name:           "logA",
					StartsWithTime: false,
				},
				{
					Name:           "logB",
					StartsWithTime: true,
				},
				{
					Name:           "logC",
					StartsWithTime: true,
				},
			}

			if strings.HasSuffix(x.srcPath, ".zip") {
				zipFile, err := os.Create(filepath.Join(currentPath, x.srcPath))
				require.NoError(t, err)
				defer zipFile.Close()

				writer := zip.NewWriter(zipFile)

				files := []struct {
					Name, Body string
				}{
					{"logA.log", "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n"},
					{"logB.log", "{\"timestamp\":\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n"},
					{"logC.log", "{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n"},
				}

				for _, file := range files {
					f, err := writer.Create(file.Name)
					require.NoError(t, err)
					_, err = f.Write([]byte(file.Body))
					require.NoError(t, err)
				}

				err = writer.Close()
				require.NoError(t, err)
			} else {
				if _, err := os.Stat(filepath.Join(currentPath, x.srcPath)); os.IsNotExist(err) {
					err = os.Mkdir(filepath.Join(currentPath, x.srcPath), os.ModePerm)
					require.NoError(t, err)
				}

				logA, err := os.Create(filepath.Join(currentPath, x.srcPath, "logA_events.log"))
				require.NoError(t, err)
				defer logA.Close()

				logB, err := os.Create(filepath.Join(currentPath, x.srcPath, "logB_events.log"))
				require.NoError(t, err)
				defer logB.Close()

				logC, err := os.Create(filepath.Join(currentPath, x.srcPath, "logC_events.log"))
				require.NoError(t, err)
				defer logC.Close()

				_, err = logA.WriteString("{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n")
				require.NoError(t, err)
				_, err = logB.WriteString("{\"timestamp\":\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n")
				require.NoError(t, err)
				_, err = logC.WriteString("{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n")
				require.NoError(t, err)
			}

			err = GetLogsFromCbcollect(filepath.Join(currentPath, x.srcPath), x.destPath)
			if x.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				logA, err := os.Open(filepath.Join(x.destPath, "logA.log"))
				require.NoError(t, err)
				logA.Close()

				logB, err := os.Open(filepath.Join(x.destPath, "logB.log"))
				require.NoError(t, err)
				logB.Close()

				logC, err := os.Open(filepath.Join(x.destPath, "logC.log"))
				require.NoError(t, err)
				logC.Close()
			}
		})
	}
}
