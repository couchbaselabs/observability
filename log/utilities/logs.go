package utilities

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/log/values"
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
		&tls.Config{InsecureSkipVerify: false, RootCAs: rootCAs})
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
