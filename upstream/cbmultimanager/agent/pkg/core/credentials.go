// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package core

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	"github.com/couchbase/tools-common/aprov"
)

func init() {
	gob.Register(&aprov.Static{})
}

func writeCredentialsToFile(file string, creds aprov.Provider) error {
	fd, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer fd.Close()
	enc := gob.NewEncoder(fd)
	if err := enc.Encode(&creds); err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}
	return nil
}

func readCredentialsFromFile(file string) (aprov.Provider, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer fd.Close()
	dec := gob.NewDecoder(fd)
	var value interface{}
	if err := dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	if provider, ok := value.(aprov.Provider); ok {
		return provider, nil
	}
	return nil, fmt.Errorf("got unexpected credentials value %T", value)
}

func removeCredentialsFile(file string) error {
	err := os.Remove(file)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("encountered error while purging credential file: %w", err)
}
