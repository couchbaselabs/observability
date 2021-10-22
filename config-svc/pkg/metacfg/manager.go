// Copyright 2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metacfg

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type ConfigManager interface {
	Get() Config
	Set(config *Config) error
}

type EphemeralConfigManager struct {
	value *Config
	mux   sync.RWMutex
}

func (m *EphemeralConfigManager) Get() Config {
	m.mux.RLock()
	val := *m.value
	m.mux.RUnlock()
	return val
}

func (m *EphemeralConfigManager) Set(val *Config) error {
	m.mux.Lock()
	m.value = val
	m.mux.Unlock()
	return nil
}

// ReadConfigFromFile reads the meta-config from a .yaml file, and returns a ConfigManager initialised with it.
// If the file doesn't exist, and allowDefault is true, returns a ConfigManager with the default values.
// In all other cases, returns an error.
//
// Note that the readOnly option is only present for future use and only `true` is currently supported.
func ReadConfigFromFile(filePath string, readOnly bool, allowDefault bool) (ConfigManager, error) {
	if !readOnly {
		return nil, fmt.Errorf("only read-only config is supported for now")
	}

	cfgFile, err := os.Open(filePath)

	if errors.Is(err, os.ErrNotExist) {
		// Config file doesn't exist
		if allowDefault {
			// But we can use the defaults
			return ReadOnlyConfigManager{value: NewDefault()}, nil
		}
		return nil, fmt.Errorf("config file doesn't exist and defaults disabled")
	}
	if err != nil {
		// File exists, but we can't access it
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}

	// Config file exists, and we can open it, try reading it
	defer cfgFile.Close()
	data, err := io.ReadAll(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read from configuration file: %w", err)
	}
	initialValue, err := FromYAMLValidate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration file: %w", err)
	}

	return ReadOnlyConfigManager{value: initialValue}, nil
}

type ReadOnlyConfigManager struct {
	value *Config
}

func (f ReadOnlyConfigManager) Get() Config {
	return *f.value
}

func (f ReadOnlyConfigManager) Set(_ *Config) error {
	return fmt.Errorf("cannot set config for %T", f)
}
