// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package utilities

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

// GetLog gets a SASL or diag log and creates the file in the current working directory.
func GetLog(endpoint string, outputFile string, cred *values.Credentials) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	client, err := couchbase.NewClient([]string{cred.Cluster}, cred.User, cred.Password,
		&tls.Config{InsecureSkipVerify: false, RootCAs: rootCAs}, true)
	if err != nil {
		return err
	}

	var resp io.ReadCloser
	if endpoint == "diag" {
		resp, err = client.GetDiagLog(ctx)
	} else {
		resp, err = client.GetSASLLogs(ctx, endpoint)
	}

	if err != nil {
		return err
	}

	defer resp.Close()

	_, err = io.Copy(f, resp)
	return err
}
