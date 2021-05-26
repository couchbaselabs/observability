package utilities

import (
	"archive/zip"
	"os"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/parsers"

	"github.com/stretchr/testify/require"
)

type unzipTestCase struct {
	name          string
	destPath      string
	srcPath       string
	errorExpected bool
}

func TestUnzip(t *testing.T) {
	currentPath, err := os.Getwd()
	require.NoError(t, err)

	err = os.MkdirAll(currentPath+"/newSrcFolder", os.ModePerm)
	require.NoError(t, err)

	testCases := []unzipTestCase{
		{
			name:     "currentDestPath",
			destPath: currentPath,
			srcPath:  "zipped_file.zip",
		},
		{
			name:     "newDestPath",
			destPath: currentPath + "/newFolder",
			srcPath:  "zipped_file.zip",
		},
		{
			name:          "emptyDestPath",
			destPath:      "",
			srcPath:       "zipped_file.zip",
			errorExpected: true,
		},
		{
			name:     "newSourcePath",
			destPath: currentPath,
			srcPath:  currentPath + "/newSrcFolder/zipped_file.zip",
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
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

			zipFile, err := os.Create(x.srcPath)
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

			err = Unzip("zipped_file.zip", x.destPath)
			if x.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				logA, err := os.Open(x.destPath + "/logA.log")
				require.NoError(t, err)
				logA.Close()

				logB, err := os.Open(x.destPath + "/logB.log")
				require.NoError(t, err)
				logB.Close()

				logC, err := os.Open(x.destPath + "/logC.log")
				require.NoError(t, err)
				logC.Close()
			}
		})
	}
}
